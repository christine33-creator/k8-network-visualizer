package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/christine33-creator/k8-network-visualizer/pkg/analyzer"
	"github.com/christine33-creator/k8-network-visualizer/pkg/collector"
	"github.com/christine33-creator/k8-network-visualizer/pkg/graph"
	"github.com/christine33-creator/k8-network-visualizer/pkg/k8s"
	"github.com/christine33-creator/k8-network-visualizer/pkg/prober"
	corev1 "k8s.io/api/core/v1"
)

// DTO structures for API responses
type PodDTO struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"`
	NodeName  string            `json:"node_name"`
	PodIP     string            `json:"pod_ip"`
	Labels    map[string]string `json:"labels"`
	Ready     bool              `json:"ready,omitempty"`
	Restarts  int32             `json:"restarts,omitempty"`
}

type ServiceDTO struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	ClusterIP string            `json:"cluster_ip"`
	Ports     []PortDTO         `json:"ports"`
	Labels    map[string]string `json:"labels"`
}

type PortDTO struct {
	Port       int32  `json:"port"`
	TargetPort int32  `json:"target_port"`
	Protocol   string `json:"protocol"`
}

type NodeDTO struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Roles      []string          `json:"roles"`
	Version    string            `json:"version"`
	InternalIP string            `json:"internal_ip"`
	Labels     map[string]string `json:"labels"`
}

var (
	kubeconfig     = flag.String("kubeconfig", "", "Path to kubeconfig file")
	addr           = flag.String("addr", ":8080", "The address to listen on for HTTP requests")
	probeInterval  = flag.Duration("probe-interval", 30*time.Second, "Interval between connectivity probes")
	enableWebUI    = flag.Bool("enable-ui", true, "Enable web UI")
	namespace      = flag.String("namespace", "", "Namespace to watch (empty for all namespaces)")
)

func main() {
	flag.Parse()

	// Set up logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting Kubernetes Network Visualizer...")

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient(*kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	log.Println("Successfully connected to Kubernetes cluster")

	// Initialize components
	networkCollector := collector.NewCollector(k8sClient, *namespace)
	networkProber := prober.NewProber(k8sClient)
	graphEngine := graph.NewEngine()
	networkAnalyzer := analyzer.NewAnalyzer(graphEngine)

	// Start data collection
	log.Println("Starting data collection...")
	go networkCollector.Start(ctx)

	// Start probing
	log.Printf("Starting connectivity probes (interval: %v)...", *probeInterval)
	go networkProber.StartProbing(ctx, *probeInterval)

	// Start analysis engine
	log.Println("Starting analysis engine...")
	go networkAnalyzer.Start(ctx, networkCollector, networkProber)

	// Set up HTTP server
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/topology", topologyHandler(graphEngine))
	mux.HandleFunc("/api/nodes", nodesHandler(networkCollector))
	mux.HandleFunc("/api/pods", podsHandler(networkCollector))
	mux.HandleFunc("/api/services", servicesHandler(networkCollector))
	mux.HandleFunc("/api/policies", policiesHandler(networkCollector))
	mux.HandleFunc("/api/probes", probesHandler(networkProber))
	mux.HandleFunc("/api/issues", issuesHandler(networkAnalyzer))
	mux.HandleFunc("/api/simulate", simulateHandler(networkAnalyzer))
	
	// WebSocket endpoint for real-time updates
	mux.HandleFunc("/ws", websocketHandler(graphEngine))

	// Serve static files for web UI
	if *enableWebUI {
		fs := http.FileServer(http.Dir("./frontend/build"))
		mux.Handle("/", fs)
	}

	// Start HTTP server
	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	log.Printf("Server starting on %s", *addr)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status":"healthy"}`)
}

func topologyHandler(engine *graph.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		topology := engine.GetTopology()
		if err := json.NewEncoder(w).Encode(topology); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func nodesHandler(collector *collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		nodes := collector.GetNodes()
		
		// Convert to DTOs
		nodeDTOs := make([]NodeDTO, 0, len(nodes))
		for _, node := range nodes {
			nodeDTO := NodeDTO{
				Name:    node.Name,
				Labels:  node.Labels,
				Version: node.Status.NodeInfo.KubeletVersion,
			}
			
			// Get node status
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady {
					if condition.Status == corev1.ConditionTrue {
						nodeDTO.Status = "Ready"
					} else {
						nodeDTO.Status = "NotReady"
					}
					break
				}
			}
			
			// Get roles from labels
			roles := make([]string, 0)
			for key := range node.Labels {
				if strings.HasPrefix(key, "node-role.kubernetes.io/") {
					role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
					if role != "" {
						roles = append(roles, role)
					}
				}
			}
			nodeDTO.Roles = roles
			
			// Get internal IP
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeInternalIP {
					nodeDTO.InternalIP = address.Address
					break
				}
			}
			
			nodeDTOs = append(nodeDTOs, nodeDTO)
		}
		
		if err := json.NewEncoder(w).Encode(nodeDTOs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func podsHandler(collector *collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		pods := collector.GetPods()
		
		// Convert to DTOs
		podDTOs := make([]PodDTO, 0, len(pods))
		for _, pod := range pods {
			podDTO := PodDTO{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
				NodeName:  pod.Spec.NodeName,
				PodIP:     pod.Status.PodIP,
				Labels:    pod.Labels,
			}
			
			// Calculate ready status
			podDTO.Ready = isPodReady(pod)
			
			// Calculate restart count
			podDTO.Restarts = getPodRestartCount(pod)
			
			podDTOs = append(podDTOs, podDTO)
		}
		
		if err := json.NewEncoder(w).Encode(podDTOs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func servicesHandler(collector *collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		services := collector.GetServices()
		
		// Convert to DTOs
		serviceDTOs := make([]ServiceDTO, 0, len(services))
		for _, service := range services {
			serviceDTO := ServiceDTO{
				Name:      service.Name,
				Namespace: service.Namespace,
				Type:      string(service.Spec.Type),
				ClusterIP: service.Spec.ClusterIP,
				Labels:    service.Labels,
				Ports:     make([]PortDTO, len(service.Spec.Ports)),
			}
			
			// Convert ports
			for i, port := range service.Spec.Ports {
				serviceDTO.Ports[i] = PortDTO{
					Port:       port.Port,
					TargetPort: port.TargetPort.IntVal,
					Protocol:   string(port.Protocol),
				}
			}
			
			serviceDTOs = append(serviceDTOs, serviceDTO)
		}
		
		if err := json.NewEncoder(w).Encode(serviceDTOs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func policiesHandler(collector *collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		policies := collector.GetNetworkPolicies()
		if err := json.NewEncoder(w).Encode(policies); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func probesHandler(prober *prober.Prober) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		results := prober.GetResults()
		if err := json.NewEncoder(w).Encode(results); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func issuesHandler(analyzer *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		issues := analyzer.GetIssues()
		if err := json.NewEncoder(w).Encode(issues); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func simulateHandler(analyzer *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// TODO: Implement simulation logic
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"simulation not yet implemented"}`)
	}
}

func websocketHandler(engine *graph.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement WebSocket handler for real-time updates
		http.Error(w, "WebSocket not yet implemented", http.StatusNotImplemented)
	}
}

// Helper functions for pod status calculation
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getPodRestartCount(pod *corev1.Pod) int32 {
	var totalRestarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		totalRestarts += containerStatus.RestartCount
	}
	return totalRestarts
}