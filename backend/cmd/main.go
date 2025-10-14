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
	"github.com/christine33-creator/k8-network-visualizer/pkg/simulator"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"io"
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
	networkSimulator := simulator.NewSimulator(graphEngine)

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
	mux.HandleFunc("/api/insights", insightsHandler(networkAnalyzer))
	mux.HandleFunc("/api/simulate", simulateHandler(networkSimulator, networkCollector))
	mux.HandleFunc("/api/simulations", simulationsHandler(networkAnalyzer))
	
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

func simulateHandler(sim *simulator.Simulator, collector *collector.Collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse simulation type from query params
		simType := r.URL.Query().Get("type")
		if simType == "" {
			simType = "network_policy"
		}

		w.Header().Set("Content-Type", "application/json")

		switch simType {
		case "network_policy":
			// Parse NetworkPolicy from body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var policy networkingv1.NetworkPolicy
			if err := yaml.Unmarshal(body, &policy); err != nil {
				http.Error(w, "Invalid NetworkPolicy YAML: "+err.Error(), http.StatusBadRequest)
				return
			}

			// Update simulator with current resources
			sim.UpdateResources(collector.GetPods(), collector.GetServices(), collector.GetNetworkPolicies())

			// Run simulation
			result, err := sim.SimulateNetworkPolicy(&policy, "apply")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

		case "pod_failure":
			podName := r.URL.Query().Get("pod")
			namespace := r.URL.Query().Get("namespace")
			if podName == "" || namespace == "" {
				http.Error(w, "Missing pod or namespace parameter", http.StatusBadRequest)
				return
			}

			// Update simulator with current resources
			sim.UpdateResources(collector.GetPods(), collector.GetServices(), collector.GetNetworkPolicies())

			result, err := sim.SimulatePodFailure(podName, namespace)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

		case "node_failure":
			nodeName := r.URL.Query().Get("node")
			if nodeName == "" {
				http.Error(w, "Missing node parameter", http.StatusBadRequest)
				return
			}

			// Update simulator with current resources
			sim.UpdateResources(collector.GetPods(), collector.GetServices(), collector.GetNetworkPolicies())

			result, err := sim.SimulateNodeFailure(nodeName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

		default:
			http.Error(w, "Unknown simulation type", http.StatusBadRequest)
		}
	}
			Changes     map[string]interface{} `json:"changes"`
			Scope       []string               `json:"scope"`
			Description string                 `json:"description"`
		}
		
		var request SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// For now, return a mock response since the full simulation isn't implemented
		result := map[string]interface{}{
			"id": "sim-" + request.Type + "-001",
			"request": request,
			"connectivity_impact": []map[string]interface{}{
				{
					"area": "Pod-to-Pod Communication",
					"description": "May affect communication patterns",
					"risk": "medium",
					"likelihood": 0.7,
				},
			},
			"security_impact": []map[string]interface{}{
				{
					"area": "Attack Surface",
					"description": "Potential security implications",
					"risk": "low",
					"likelihood": 0.5,
				},
			},
			"performance_impact": []map[string]interface{}{
				{
					"area": "Response Time",
					"description": "Minor performance considerations",
					"risk": "low",
					"likelihood": 0.3,
				},
			},
			"overall_risk": "medium",
			"confidence": 0.85,
			"timestamp": time.Now().Format(time.RFC3339),
		}
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func insightsHandler(analyzer *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		insights := analyzer.GetInsights()
		if err := json.NewEncoder(w).Encode(insights); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func simulationsHandler(analyzer *analyzer.Analyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		simulations := analyzer.GetSimulations()
		if err := json.NewEncoder(w).Encode(simulations); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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