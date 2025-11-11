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

	"github.com/christine33-creator/k8-network-visualizer/pkg/ai"
	"github.com/christine33-creator/k8-network-visualizer/pkg/analyzer"
	"github.com/christine33-creator/k8-network-visualizer/pkg/collector"
	"github.com/christine33-creator/k8-network-visualizer/pkg/correlation"
	"github.com/christine33-creator/k8-network-visualizer/pkg/graph"
	"github.com/christine33-creator/k8-network-visualizer/pkg/k8s"
	"github.com/christine33-creator/k8-network-visualizer/pkg/prober"
	"github.com/christine33-creator/k8-network-visualizer/pkg/simulator"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	prometheusURL  = flag.String("prometheus-url", "http://localhost:9090", "Prometheus server URL for metrics collection")
	aiAPIKey       = flag.String("ai-api-key", "", "OpenRouter AI API key for enhanced analysis")
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

	// Initialize AI client if API key provided (check both flag and environment variable)
	apiKey := *aiAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	
	if apiKey != "" {
		log.Println("Initializing AI client for enhanced analysis...")
		aiClient := ai.NewClient(apiKey)
		networkSimulator.SetAIClient(aiClient)
		log.Println("AI client initialized successfully")
	} else {
		log.Println("AI API key not provided - simulations will use rule-based analysis only")
	}

	// Initialize correlation engine
	log.Println("Initializing correlation engine...")
	correlationEngine, metricsCollector, err := setupCorrelationEngine(*prometheusURL)
	if err != nil {
		log.Printf("Warning: Failed to initialize correlation engine: %v", err)
		log.Println("Continuing without correlation features...")
		correlationEngine = nil
		metricsCollector = nil
	} else {
		log.Println("Correlation engine initialized successfully")
	}

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
	
	// Correlation engine endpoints
	if correlationEngine != nil {
		mux.HandleFunc("/api/correlations", correlationsHandler(correlationEngine))
		mux.HandleFunc("/api/correlations/details", correlationDetailsHandler(correlationEngine))
		mux.HandleFunc("/api/correlations/root-cause", rootCauseAnalysisHandler(correlationEngine))
		mux.HandleFunc("/api/metrics/ingest", ingestMetricHandler(correlationEngine))
		mux.HandleFunc("/api/dashboard/unified", unifiedDashboardHandler(correlationEngine, metricsCollector))
		log.Println("Correlation API endpoints registered")
	}
	
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
		
		// Define the struct locally to avoid import issues
		type SimulationRequest struct {
			Type        string                 `json:"type"`
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Namespace   string                 `json:"namespace"`
			Changes     map[string]interface{} `json:"changes"`
			Scope       []string               `json:"scope"`
			Parameters  map[string]interface{} `json:"parameters"`
		}
		
		var request SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Update simulator with current cluster state
		pods := collector.GetPods()
		services := collector.GetServices()
		policies := collector.GetNetworkPolicies()
		sim.UpdateResources(pods, services, policies)
		
		var result interface{}
		var err error
		
		// Run appropriate simulation based on type
		switch request.Type {
		case "policy":
			// Create a mock network policy for simulation based on description
			policy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      request.Name,
					Namespace: request.Namespace,
				},
				Spec: networkingv1.NetworkPolicySpec{
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
				},
			}
			
			// Parse description to configure policy appropriately
			description := strings.ToLower(request.Description)
			if strings.Contains(description, "block") || strings.Contains(description, "deny") {
				// No egress rules = deny all
				policy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{}
			} else if strings.Contains(description, "dns") {
				// Allow DNS only
				policy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
					{
						Ports: []networkingv1.NetworkPolicyPort{
							{Protocol: func() *corev1.Protocol { p := corev1.ProtocolUDP; return &p }(), Port: func() *intstr.IntOrString { p := intstr.FromInt(53); return &p }()},
						},
					},
				}
			}
			
			result, err = sim.SimulateNetworkPolicy(policy, "add")
			
		case "resource":
			// Simulate pod failure or scaling
			if podName, ok := request.Parameters["pod_name"].(string); ok {
				result, err = sim.SimulatePodFailure(podName, request.Namespace)
			} else {
				err = fmt.Errorf("missing pod_name parameter for resource simulation")
			}
			
		case "topology":
			// Simulate node failure
			if nodeName, ok := request.Parameters["node_name"].(string); ok {
				result, err = sim.SimulateNodeFailure(nodeName)
			} else {
				err = fmt.Errorf("missing node_name parameter for topology simulation")
			}
			
		default:
			http.Error(w, "Invalid simulation type", http.StatusBadRequest)
			return
		}
		
		if err != nil {
			http.Error(w, fmt.Sprintf("Simulation failed: %v", err), http.StatusInternalServerError)
			return
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

// Correlation Engine handlers

func setupCorrelationEngine(prometheusURL string) (*correlation.CorrelationEngine, *correlation.MetricsCollector, error) {
	// Create correlation engine with 5-minute time window
	engine := correlation.NewCorrelationEngine(5 * time.Minute)

	// Create metrics collector
	collector, err := correlation.NewMetricsCollector(prometheusURL, engine)
	if err != nil {
		return nil, nil, err
	}

	// Start collecting metrics
	ctx := context.Background()
	go collector.Start(ctx)

	return engine, collector, nil
}

func correlationsHandler(engine *correlation.CorrelationEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlations := engine.GetCorrelations()
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(correlations); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func correlationDetailsHandler(engine *correlation.CorrelationEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.URL.Query().Get("id")
		if correlationID == "" {
			http.Error(w, "Missing correlation ID", http.StatusBadRequest)
			return
		}

		correlation := engine.GetCorrelationByID(correlationID)
		if correlation == nil {
			http.Error(w, "Correlation not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(correlation); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func ingestMetricHandler(engine *correlation.CorrelationEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Type   string             `json:"type"`
			Name   string             `json:"name"`
			Value  float64            `json:"value"`
			Labels map[string]string  `json:"labels"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var metricType correlation.MetricType
		switch request.Type {
		case "network":
			metricType = correlation.MetricTypeNetwork
		case "application":
			metricType = correlation.MetricTypeApplication
		case "infrastructure":
			metricType = correlation.MetricTypeInfrastructure
		default:
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
			return
		}

		engine.IngestMetric(metricType, request.Name, request.Value, request.Labels)

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}
}

func unifiedDashboardHandler(engine *correlation.CorrelationEngine, collector *correlation.MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		// Get all correlations
		correlations := engine.GetCorrelations()

		// Get specific correlation scenarios
		packetLossCorrs := collector.GetPacketLossCorrelations(ctx)
		nodeHealthImpact := collector.GetNodeHealthImpact(ctx)
		cniImpact := collector.GetCNIPerformanceImpact(ctx)

		dashboard := struct {
			Timestamp           time.Time                    `json:"timestamp"`
			OverallStatus       string                       `json:"overall_status"`
			AllCorrelations     []correlation.Correlation    `json:"all_correlations"`
			PacketLossCorrs     []correlation.Correlation    `json:"packet_loss_correlations"`
			NodeHealthImpact    []correlation.Correlation    `json:"node_health_impact"`
			CNIImpact           []correlation.Correlation    `json:"cni_impact"`
			CriticalAlerts      []CriticalAlert              `json:"critical_alerts"`
			HealthScore         HealthScore                  `json:"health_score"`
		}{
			Timestamp:           time.Now(),
			OverallStatus:       calculateOverallStatus(correlations),
			AllCorrelations:     correlations,
			PacketLossCorrs:     packetLossCorrs,
			NodeHealthImpact:    nodeHealthImpact,
			CNIImpact:           cniImpact,
			CriticalAlerts:      extractCriticalAlertsTyped(correlations),
			HealthScore:         calculateHealthScoreTyped(correlations),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(dashboard); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func rootCauseAnalysisHandler(engine *correlation.CorrelationEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.URL.Query().Get("id")
		if correlationID == "" {
			http.Error(w, "Missing correlation ID", http.StatusBadRequest)
			return
		}

		correlation := engine.GetCorrelationByID(correlationID)
		if correlation == nil {
			http.Error(w, "Correlation not found", http.StatusNotFound)
			return
		}

		if correlation.RootCause == nil {
			http.Error(w, "No root cause analysis available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(correlation.RootCause); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func calculateOverallStatus(correlations []correlation.Correlation) string {
	criticalCount := 0
	highCount := 0

	for _, corr := range correlations {
		switch corr.Impact.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
	}

	if criticalCount > 0 {
		return "critical"
	} else if highCount > 2 {
		return "degraded"
	} else if highCount > 0 {
		return "warning"
	}

	return "healthy"
}

type CriticalAlert struct {
	Severity      string    `json:"severity"`
	Message       string    `json:"message"`
	Source        string    `json:"source"`
	Timestamp     time.Time `json:"timestamp"`
	CorrelationID string    `json:"correlation_id"`
	Impact        string    `json:"impact"`
}

type HealthScore struct {
	Overall          int                `json:"overall"`
	Network          int                `json:"network"`
	Application      int                `json:"application"`
	Infrastructure   int                `json:"infrastructure"`
	TrendDirection   string             `json:"trend_direction"`
	Details          map[string]float64 `json:"details"`
}

func extractCriticalAlertsTyped(correlations []correlation.Correlation) []CriticalAlert {
	alerts := []CriticalAlert{}

	for _, corr := range correlations {
		if corr.Impact.Severity == "critical" || corr.Impact.Severity == "high" {
			message := "Correlation detected between " + corr.PrimaryMetric.Name
			if len(corr.RelatedMetrics) > 0 {
				message += " and " + corr.RelatedMetrics[0].Name
			}
			if corr.RootCause != nil {
				message = corr.RootCause.Cause
			}

			alert := CriticalAlert{
				Severity:      corr.Impact.Severity,
				Message:       message,
				Source:        corr.PrimaryMetric.Name,
				Timestamp:     corr.TimeDetected,
				CorrelationID: corr.ID,
				Impact:        corr.Impact.BusinessImpact,
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func calculateHealthScoreTyped(correlations []correlation.Correlation) HealthScore {
	score := HealthScore{
		Overall:        100,
		Network:        100,
		Application:    100,
		Infrastructure: 100,
		TrendDirection: "stable",
		Details:        make(map[string]float64),
	}

	// Deduct points based on severity
	for _, corr := range correlations {
		deduction := 0
		switch corr.Impact.Severity {
		case "critical":
			deduction = 30
		case "high":
			deduction = 15
		case "medium":
			deduction = 5
		}

		score.Overall -= deduction

		// Deduct from specific categories
		switch corr.PrimaryMetric.Type {
		case correlation.MetricTypeNetwork:
			score.Network -= deduction
		case correlation.MetricTypeApplication:
			score.Application -= deduction
		case correlation.MetricTypeInfrastructure:
			score.Infrastructure -= deduction
		}
	}

	// Ensure scores don't go below 0
	if score.Overall < 0 {
		score.Overall = 0
	}
	if score.Network < 0 {
		score.Network = 0
	}
	if score.Application < 0 {
		score.Application = 0
	}
	if score.Infrastructure < 0 {
		score.Infrastructure = 0
	}

	// Determine trend
	if score.Overall < 60 {
		score.TrendDirection = "degrading"
	} else if score.Overall > 85 {
		score.TrendDirection = "improving"
	}

	// Add detailed metrics
	totalErrors := 0.0
	totalLatency := 0.0
	totalAvail := 0.0
	errorCount := 0
	latencyCount := 0
	availCount := 0

	for _, corr := range correlations {
		if corr.Impact.ErrorRate > 0 {
			totalErrors += corr.Impact.ErrorRate
			errorCount++
		}
		if corr.Impact.Latency > 0 {
			totalLatency += corr.Impact.Latency
			latencyCount++
		}
		if corr.Impact.Availability > 0 {
			totalAvail += corr.Impact.Availability
			availCount++
		}
	}

	if errorCount > 0 {
		score.Details["error_rate"] = totalErrors / float64(errorCount)
	} else {
		score.Details["error_rate"] = 0.0
	}
	
	if latencyCount > 0 {
		score.Details["latency_impact"] = totalLatency / float64(latencyCount)
	} else {
		score.Details["latency_impact"] = 0.0
	}
	
	if availCount > 0 {
		score.Details["availability"] = totalAvail / float64(availCount)
	} else {
		score.Details["availability"] = 100.0
	}

	return score
}

// Legacy functions for backward compatibility (can be removed later)
func extractCriticalAlerts(correlations []correlation.Correlation) []interface{} {
	alerts := []interface{}{}

	for _, corr := range correlations {
		if corr.Impact.Severity == "critical" || corr.Impact.Severity == "high" {
			message := "Correlation detected between " + corr.PrimaryMetric.Name
			if len(corr.RelatedMetrics) > 0 {
				message += " and " + corr.RelatedMetrics[0].Name
			}
			if corr.RootCause != nil {
				message = corr.RootCause.Cause
			}

			alert := map[string]interface{}{
				"severity":       corr.Impact.Severity,
				"message":        message,
				"source":         corr.PrimaryMetric.Name,
				"timestamp":      corr.TimeDetected,
				"correlation_id": corr.ID,
				"impact":         corr.Impact.BusinessImpact,
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func calculateHealthScore(correlations []correlation.Correlation) interface{} {
	score := map[string]interface{}{
		"overall":          100,
		"network":          100,
		"application":      100,
		"infrastructure":   100,
		"trend_direction":  "stable",
		"details":          map[string]float64{},
	}

	overall := 100
	network := 100
	application := 100
	infrastructure := 100

	// Deduct points based on severity
	for _, corr := range correlations {
		deduction := 0
		switch corr.Impact.Severity {
		case "critical":
			deduction = 30
		case "high":
			deduction = 15
		case "medium":
			deduction = 5
		}

		overall -= deduction

		// Deduct from specific categories
		switch corr.PrimaryMetric.Type {
		case correlation.MetricTypeNetwork:
			network -= deduction
		case correlation.MetricTypeApplication:
			application -= deduction
		case correlation.MetricTypeInfrastructure:
			infrastructure -= deduction
		}
	}

	// Ensure scores don't go below 0
	if overall < 0 {
		overall = 0
	}
	if network < 0 {
		network = 0
	}
	if application < 0 {
		application = 0
	}
	if infrastructure < 0 {
		infrastructure = 0
	}

	score["overall"] = overall
	score["network"] = network
	score["application"] = application
	score["infrastructure"] = infrastructure

	// Determine trend
	if overall < 60 {
		score["trend_direction"] = "degrading"
	} else if overall > 85 {
		score["trend_direction"] = "improving"
	}

	// Add detailed metrics
	details := score["details"].(map[string]float64)
	details["error_rate"] = 0.0
	details["latency_impact"] = 0.0
	details["availability"] = 100.0

	if len(correlations) > 0 {
		totalErrors := 0.0
		totalLatency := 0.0
		totalAvail := 0.0
		errorCount := 0
		latencyCount := 0
		availCount := 0

		for _, corr := range correlations {
			if corr.Impact.ErrorRate > 0 {
				totalErrors += corr.Impact.ErrorRate
				errorCount++
			}
			if corr.Impact.Latency > 0 {
				totalLatency += corr.Impact.Latency
				latencyCount++
			}
			if corr.Impact.Availability > 0 {
				totalAvail += corr.Impact.Availability
				availCount++
			}
		}

		if errorCount > 0 {
			details["error_rate"] = totalErrors / float64(errorCount)
		}
		if latencyCount > 0 {
			details["latency_impact"] = totalLatency / float64(latencyCount)
		}
		if availCount > 0 {
			details["availability"] = totalAvail / float64(availCount)
		}
	}

	return score
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