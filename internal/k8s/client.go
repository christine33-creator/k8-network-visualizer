package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/christine33-creator/k8-network-visualizer/pkg/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes client
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to kubeconfig
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
	}

	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

// GetNetworkTopology discovers and returns the network topology
func (c *Client) GetNetworkTopology(ctx context.Context, namespace string) (*models.NetworkTopology, error) {
	topology := &models.NetworkTopology{
		Timestamp: time.Now(),
	}

	// Get nodes
	nodes, err := c.getNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	topology.Nodes = nodes

	// Get pods
	pods, err := c.getPods(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}
	topology.Pods = pods

	// Get services
	services, err := c.getServices(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	topology.Services = services

	// Get network policies
	policies, err := c.getNetworkPolicies(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get network policies: %w", err)
	}
	topology.Policies = policies

	// Analyze connections
	connections := c.analyzeConnections(topology)
	topology.Connections = connections

	return topology, nil
}

func (c *Client) getNodes(ctx context.Context) ([]models.Node, error) {
	nodeList, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var nodes []models.Node
	for _, node := range nodeList.Items {
		n := models.Node{
			Name:   node.Name,
			Labels: node.Labels,
			Ready:  isNodeReady(&node),
			CIDRs:  []string{node.Spec.PodCIDR},
		}

		// Get node IP
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				n.IP = addr.Address
				break
			}
		}

		nodes = append(nodes, n)
	}

	return nodes, nil
}

func (c *Client) getPods(ctx context.Context, namespace string) ([]models.Pod, error) {
	listOptions := metav1.ListOptions{}
	podList, err := c.clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	var pods []models.Pod
	for _, pod := range podList.Items {
		p := models.Pod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			IP:        pod.Status.PodIP,
			Node:      pod.Spec.NodeName,
			Labels:    pod.Labels,
			Status:    string(pod.Status.Phase),
		}

		// Extract ports
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				p.Ports = append(p.Ports, models.Port{
					Name:     port.Name,
					Port:     port.ContainerPort,
					Protocol: string(port.Protocol),
				})
			}
		}

		pods = append(pods, p)
	}

	return pods, nil
}

func (c *Client) getServices(ctx context.Context, namespace string) ([]models.Service, error) {
	listOptions := metav1.ListOptions{}
	serviceList, err := c.clientset.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	var services []models.Service
	for _, svc := range serviceList.Items {
		s := models.Service{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Type:      string(svc.Spec.Type),
			ClusterIP: svc.Spec.ClusterIP,
			Selector:  svc.Spec.Selector,
		}

		// Extract ports
		for _, port := range svc.Spec.Ports {
			s.Ports = append(s.Ports, models.Port{
				Name:     port.Name,
				Port:     port.Port,
				Protocol: string(port.Protocol),
			})
		}

		// Get endpoints
		endpoints, err := c.clientset.CoreV1().Endpoints(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})
		if err == nil {
			for _, subset := range endpoints.Subsets {
				for _, addr := range subset.Addresses {
					s.Endpoints = append(s.Endpoints, addr.IP)
				}
			}
		}

		services = append(services, s)
	}

	return services, nil
}

func (c *Client) getNetworkPolicies(ctx context.Context, namespace string) ([]models.NetworkPolicy, error) {
	listOptions := metav1.ListOptions{}
	policyList, err := c.clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	var policies []models.NetworkPolicy
	for _, policy := range policyList.Items {
		p := models.NetworkPolicy{
			Name:      policy.Name,
			Namespace: policy.Namespace,
			Selector:  policy.Spec.PodSelector.MatchLabels,
		}

		// Convert ingress rules
		for _, rule := range policy.Spec.Ingress {
			r := models.Rule{}
			for _, from := range rule.From {
				selector := models.Selector{}
				if from.PodSelector != nil {
					selector.PodSelector = from.PodSelector.MatchLabels
				}
				if from.NamespaceSelector != nil {
					selector.NamespaceSelector = from.NamespaceSelector.MatchLabels
				}
				if from.IPBlock != nil {
					selector.IPBlock = &models.IPBlock{
						CIDR:   from.IPBlock.CIDR,
						Except: from.IPBlock.Except,
					}
				}
				r.From = append(r.From, selector)
			}

			for _, port := range rule.Ports {
				r.Ports = append(r.Ports, models.Port{
					Port:     port.Port.IntVal,
					Protocol: string(*port.Protocol),
				})
			}

			p.Ingress = append(p.Ingress, r)
		}

		// Convert egress rules
		for _, rule := range policy.Spec.Egress {
			r := models.Rule{}
			for _, to := range rule.To {
				selector := models.Selector{}
				if to.PodSelector != nil {
					selector.PodSelector = to.PodSelector.MatchLabels
				}
				if to.NamespaceSelector != nil {
					selector.NamespaceSelector = to.NamespaceSelector.MatchLabels
				}
				if to.IPBlock != nil {
					selector.IPBlock = &models.IPBlock{
						CIDR:   to.IPBlock.CIDR,
						Except: to.IPBlock.Except,
					}
				}
				r.To = append(r.To, selector)
			}

			for _, port := range rule.Ports {
				r.Ports = append(r.Ports, models.Port{
					Port:     port.Port.IntVal,
					Protocol: string(*port.Protocol),
				})
			}

			p.Egress = append(p.Egress, r)
		}

		policies = append(policies, p)
	}

	return policies, nil
}

func (c *Client) analyzeConnections(topology *models.NetworkTopology) []models.Connection {
	var connections []models.Connection

	// Analyze pod-to-pod connections through services
	for _, service := range topology.Services {
		for _, endpoint := range service.Endpoints {
			for _, port := range service.Ports {
				connection := models.Connection{
					Source:      service.ClusterIP,
					Destination: endpoint,
					Port:        port.Port,
					Protocol:    port.Protocol,
					Status:      "unknown", // Would need actual connectivity testing
				}
				connections = append(connections, connection)
			}
		}
	}

	return connections
}

func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
