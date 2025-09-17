package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client wraps the Kubernetes client
type Client struct {
	clientset kubernetes.Interface
	config    *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath != "" {
		// Use the provided kubeconfig path
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		// Use KUBECONFIG environment variable
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if home := homedir.HomeDir(); home != "" {
		// Try default kubeconfig location
		kubeconfigPath = filepath.Join(home, ".kube", "config")
		if _, statErr := os.Stat(kubeconfigPath); statErr == nil {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		} else {
			// Try in-cluster config
			config, err = rest.InClusterConfig()
		}
	} else {
		// Try in-cluster config
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Test connection
	_, err = clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// Clientset returns the Kubernetes clientset
func (c *Client) Clientset() kubernetes.Interface {
	return c.clientset
}

// Config returns the REST config
func (c *Client) Config() *rest.Config {
	return c.config
}