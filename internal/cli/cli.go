package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/christine33-creator/k8-network-visualizer/internal/config"
	"github.com/christine33-creator/k8-network-visualizer/internal/k8s"
	"github.com/christine33-creator/k8-network-visualizer/internal/visualizer"
	"github.com/christine33-creator/k8-network-visualizer/pkg/models"
)

// Run executes the main application logic
func Run(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(cfg.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get network topology
	ctx := context.Background()
	topology, err := client.GetNetworkTopology(ctx, cfg.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get network topology: %w", err)
	}

	// Output based on format
	switch cfg.Output {
	case "json":
		return outputJSON(topology)
	case "web":
		return serveWeb(topology, cfg.Port)
	default: // "cli"
		return outputCLI(topology)
	}
}

func outputJSON(topology *models.NetworkTopology) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(topology)
}

func outputCLI(topology *models.NetworkTopology) error {
	fmt.Printf("Kubernetes Network Topology - %s\n", topology.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 50))

	// Nodes
	fmt.Printf("\nNodes (%d):\n", len(topology.Nodes))
	for _, node := range topology.Nodes {
		status := "Not Ready"
		if node.Ready {
			status = "Ready"
		}
		fmt.Printf("  ├─ %s (%s) - %s\n", node.Name, node.IP, status)
		if len(node.CIDRs) > 0 {
			fmt.Printf("     └─ CIDR: %s\n", strings.Join(node.CIDRs, ", "))
		}
	}

	// Pods
	fmt.Printf("\nPods (%d):\n", len(topology.Pods))
	for _, pod := range topology.Pods {
		fmt.Printf("  ├─ %s/%s (%s) on %s - %s\n",
			pod.Namespace, pod.Name, pod.IP, pod.Node, pod.Status)
		if len(pod.Ports) > 0 {
			fmt.Printf("     └─ Ports: ")
			var ports []string
			for _, port := range pod.Ports {
				ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
			}
			fmt.Printf("%s\n", strings.Join(ports, ", "))
		}
	}

	// Services
	fmt.Printf("\nServices (%d):\n", len(topology.Services))
	for _, service := range topology.Services {
		fmt.Printf("  ├─ %s/%s (%s) - %s\n",
			service.Namespace, service.Name, service.ClusterIP, service.Type)
		if len(service.Ports) > 0 {
			fmt.Printf("     ├─ Ports: ")
			var ports []string
			for _, port := range service.Ports {
				ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
			}
			fmt.Printf("%s\n", strings.Join(ports, ", "))
		}
		if len(service.Endpoints) > 0 {
			fmt.Printf("     └─ Endpoints: %s\n", strings.Join(service.Endpoints, ", "))
		}
	}

	// Network Policies
	fmt.Printf("\nNetwork Policies (%d):\n", len(topology.Policies))
	for _, policy := range topology.Policies {
		fmt.Printf("  ├─ %s/%s\n", policy.Namespace, policy.Name)
		if len(policy.Ingress) > 0 {
			fmt.Printf("     ├─ Ingress rules: %d\n", len(policy.Ingress))
		}
		if len(policy.Egress) > 0 {
			fmt.Printf("     └─ Egress rules: %d\n", len(policy.Egress))
		}
	}

	// Connections
	fmt.Printf("\nConnections (%d):\n", len(topology.Connections))
	for _, conn := range topology.Connections {
		fmt.Printf("  ├─ %s → %s:%d/%s [%s]\n",
			conn.Source, conn.Destination, conn.Port, conn.Protocol, conn.Status)
	}

	return nil
}

func serveWeb(topology *models.NetworkTopology, port int) error {
	fmt.Printf("Starting web server on port %d...\n", port)

	// Create visualizer
	viz := visualizer.New()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := viz.GenerateHTML(topology)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	http.HandleFunc("/api/topology", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(topology)
	})

	fmt.Printf("Open http://localhost:%d in your browser\n", port)
	return http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
