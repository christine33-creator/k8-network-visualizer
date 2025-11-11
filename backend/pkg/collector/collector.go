package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/christine33-creator/k8-network-visualizer/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// Collector collects Kubernetes network resources
type Collector struct {
	client    *k8s.Client
	namespace string
	
	// Data stores
	mu              sync.RWMutex
	pods            map[string]*corev1.Pod
	services        map[string]*corev1.Service
	endpoints       map[string]*corev1.Endpoints
	nodes           map[string]*corev1.Node
	networkPolicies map[string]*networkingv1.NetworkPolicy
	
	// Informers
	podInformer    cache.SharedIndexInformer
	svcInformer    cache.SharedIndexInformer
	epInformer     cache.SharedIndexInformer
	nodeInformer   cache.SharedIndexInformer
	policyInformer cache.SharedIndexInformer
}

// NewCollector creates a new resource collector
func NewCollector(client *k8s.Client, namespace string) *Collector {
	return &Collector{
		client:          client,
		namespace:       namespace,
		pods:            make(map[string]*corev1.Pod),
		services:        make(map[string]*corev1.Service),
		endpoints:       make(map[string]*corev1.Endpoints),
		nodes:           make(map[string]*corev1.Node),
		networkPolicies: make(map[string]*networkingv1.NetworkPolicy),
	}
}

// Start begins collecting resources
func (c *Collector) Start(ctx context.Context) error {
	// Set up informers for each resource type
	if err := c.setupPodInformer(ctx); err != nil {
		return fmt.Errorf("failed to setup pod informer: %w", err)
	}
	
	if err := c.setupServiceInformer(ctx); err != nil {
		return fmt.Errorf("failed to setup service informer: %w", err)
	}
	
	if err := c.setupEndpointsInformer(ctx); err != nil {
		return fmt.Errorf("failed to setup endpoints informer: %w", err)
	}
	
	if err := c.setupNodeInformer(ctx); err != nil {
		return fmt.Errorf("failed to setup node informer: %w", err)
	}
	
	if err := c.setupNetworkPolicyInformer(ctx); err != nil {
		return fmt.Errorf("failed to setup network policy informer: %w", err)
	}
	
	// Start all informers
	go c.podInformer.Run(ctx.Done())
	go c.svcInformer.Run(ctx.Done())
	go c.epInformer.Run(ctx.Done())
	go c.nodeInformer.Run(ctx.Done())
	go c.policyInformer.Run(ctx.Done())
	
	// Wait for caches to sync
	log.Println("Waiting for informer caches to sync...")
	if !cache.WaitForCacheSync(ctx.Done(),
		c.podInformer.HasSynced,
		c.svcInformer.HasSynced,
		c.epInformer.HasSynced,
		c.nodeInformer.HasSynced,
		c.policyInformer.HasSynced,
	) {
		return fmt.Errorf("failed to sync caches")
	}
	log.Println("Informer caches synced successfully")
	
	return nil
}

func (c *Collector) setupPodInformer(ctx context.Context) error {
	listWatch := cache.NewListWatchFromClient(
		c.client.Clientset().CoreV1().RESTClient(),
		"pods",
		c.namespace,
		fields.Everything(),
	)
	
	c.podInformer = cache.NewSharedIndexInformer(
		listWatch,
		&corev1.Pod{},
		time.Minute,
		cache.Indexers{},
	)
	
	c.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			c.mu.Lock()
			c.pods[fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)] = pod
			c.mu.Unlock()
			log.Printf("Pod added: %s/%s", pod.Namespace, pod.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			c.mu.Lock()
			c.pods[fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)] = pod
			c.mu.Unlock()
			log.Printf("Pod updated: %s/%s", pod.Namespace, pod.Name)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			c.mu.Lock()
			delete(c.pods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
			c.mu.Unlock()
			log.Printf("Pod deleted: %s/%s", pod.Namespace, pod.Name)
		},
	})
	
	return nil
}

func (c *Collector) setupServiceInformer(ctx context.Context) error {
	listWatch := cache.NewListWatchFromClient(
		c.client.Clientset().CoreV1().RESTClient(),
		"services",
		c.namespace,
		fields.Everything(),
	)
	
	c.svcInformer = cache.NewSharedIndexInformer(
		listWatch,
		&corev1.Service{},
		time.Minute,
		cache.Indexers{},
	)
	
	c.svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*corev1.Service)
			c.mu.Lock()
			c.services[fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)] = svc
			c.mu.Unlock()
			log.Printf("Service added: %s/%s", svc.Namespace, svc.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			svc := newObj.(*corev1.Service)
			c.mu.Lock()
			c.services[fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)] = svc
			c.mu.Unlock()
			log.Printf("Service updated: %s/%s", svc.Namespace, svc.Name)
		},
		DeleteFunc: func(obj interface{}) {
			svc := obj.(*corev1.Service)
			c.mu.Lock()
			delete(c.services, fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
			c.mu.Unlock()
			log.Printf("Service deleted: %s/%s", svc.Namespace, svc.Name)
		},
	})
	
	return nil
}

func (c *Collector) setupEndpointsInformer(ctx context.Context) error {
	listWatch := cache.NewListWatchFromClient(
		c.client.Clientset().CoreV1().RESTClient(),
		"endpoints",
		c.namespace,
		fields.Everything(),
	)
	
	c.epInformer = cache.NewSharedIndexInformer(
		listWatch,
		&corev1.Endpoints{},
		time.Minute,
		cache.Indexers{},
	)
	
	c.epInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ep := obj.(*corev1.Endpoints)
			c.mu.Lock()
			c.endpoints[fmt.Sprintf("%s/%s", ep.Namespace, ep.Name)] = ep
			c.mu.Unlock()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			ep := newObj.(*corev1.Endpoints)
			c.mu.Lock()
			c.endpoints[fmt.Sprintf("%s/%s", ep.Namespace, ep.Name)] = ep
			c.mu.Unlock()
		},
		DeleteFunc: func(obj interface{}) {
			ep := obj.(*corev1.Endpoints)
			c.mu.Lock()
			delete(c.endpoints, fmt.Sprintf("%s/%s", ep.Namespace, ep.Name))
			c.mu.Unlock()
		},
	})
	
	return nil
}

func (c *Collector) setupNodeInformer(ctx context.Context) error {
	listWatch := cache.NewListWatchFromClient(
		c.client.Clientset().CoreV1().RESTClient(),
		"nodes",
		metav1.NamespaceAll,
		fields.Everything(),
	)
	
	c.nodeInformer = cache.NewSharedIndexInformer(
		listWatch,
		&corev1.Node{},
		time.Minute,
		cache.Indexers{},
	)
	
	c.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			c.mu.Lock()
			c.nodes[node.Name] = node
			c.mu.Unlock()
			// Only log on first add, not on restarts
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			node := newObj.(*corev1.Node)
			c.mu.Lock()
			c.nodes[node.Name] = node
			c.mu.Unlock()
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			c.mu.Lock()
			delete(c.nodes, node.Name)
			c.mu.Unlock()
			log.Printf("Node deleted: %s", node.Name)
		},
	})
	
	return nil
}

func (c *Collector) setupNetworkPolicyInformer(ctx context.Context) error {
	listWatch := cache.NewListWatchFromClient(
		c.client.Clientset().NetworkingV1().RESTClient(),
		"networkpolicies",
		c.namespace,
		fields.Everything(),
	)
	
	c.policyInformer = cache.NewSharedIndexInformer(
		listWatch,
		&networkingv1.NetworkPolicy{},
		time.Minute,
		cache.Indexers{},
	)
	
	c.policyInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			policy := obj.(*networkingv1.NetworkPolicy)
			c.mu.Lock()
			c.networkPolicies[fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)] = policy
			c.mu.Unlock()
			log.Printf("NetworkPolicy added: %s/%s", policy.Namespace, policy.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			policy := newObj.(*networkingv1.NetworkPolicy)
			c.mu.Lock()
			c.networkPolicies[fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)] = policy
			c.mu.Unlock()
			log.Printf("NetworkPolicy updated: %s/%s", policy.Namespace, policy.Name)
		},
		DeleteFunc: func(obj interface{}) {
			policy := obj.(*networkingv1.NetworkPolicy)
			c.mu.Lock()
			delete(c.networkPolicies, fmt.Sprintf("%s/%s", policy.Namespace, policy.Name))
			c.mu.Unlock()
			log.Printf("NetworkPolicy deleted: %s/%s", policy.Namespace, policy.Name)
		},
	})
	
	return nil
}

// GetPods returns all collected pods
func (c *Collector) GetPods() []*corev1.Pod {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	pods := make([]*corev1.Pod, 0, len(c.pods))
	for _, pod := range c.pods {
		pods = append(pods, pod)
	}
	return pods
}

// GetServices returns all collected services
func (c *Collector) GetServices() []*corev1.Service {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	services := make([]*corev1.Service, 0, len(c.services))
	for _, svc := range c.services {
		services = append(services, svc)
	}
	return services
}

// GetEndpoints returns all collected endpoints
func (c *Collector) GetEndpoints() []*corev1.Endpoints {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	endpoints := make([]*corev1.Endpoints, 0, len(c.endpoints))
	for _, ep := range c.endpoints {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// GetNodes returns all collected nodes
func (c *Collector) GetNodes() []*corev1.Node {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	nodes := make([]*corev1.Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNetworkPolicies returns all collected network policies
func (c *Collector) GetNetworkPolicies() []*networkingv1.NetworkPolicy {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	policies := make([]*networkingv1.NetworkPolicy, 0, len(c.networkPolicies))
	for _, policy := range c.networkPolicies {
		policies = append(policies, policy)
	}
	return policies
}