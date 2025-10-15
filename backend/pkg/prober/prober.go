package prober

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/christine33-creator/k8-network-visualizer/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// ProbeResult represents the result of a connectivity probe
type ProbeResult struct {
	Timestamp   time.Time `json:"timestamp"`
	SourcePod   string    `json:"source_pod"`
	SourceNS    string    `json:"source_namespace"`
	TargetPod   string    `json:"target_pod,omitempty"`
	TargetNS    string    `json:"target_namespace,omitempty"`
	TargetSvc   string    `json:"target_service,omitempty"`
	TargetIP    string    `json:"target_ip"`
	TargetPort  int32     `json:"target_port"`
	ProbeType   string    `json:"probe_type"` // TCP, HTTP, gRPC
	Success     bool      `json:"success"`
	Latency     int64     `json:"latency_ms"`
	PacketLoss  float64   `json:"packet_loss"` // Percentage of packet loss
	Error       string    `json:"error,omitempty"`
	StatusCode  int       `json:"status_code,omitempty"` // For HTTP probes
}

// Prober performs connectivity probes between pods
type Prober struct {
	client  *k8s.Client
	results []ProbeResult
	mu      sync.RWMutex
}

// NewProber creates a new prober instance
func NewProber(client *k8s.Client) *Prober {
	return &Prober{
		client:  client,
		results: make([]ProbeResult, 0),
	}
}

// StartProbing starts the probing process
func (p *Prober) StartProbing(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial probe immediately
	p.runProbes(ctx)

	for {
		select {
		case <-ticker.C:
			p.runProbes(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// runProbes executes all probes
func (p *Prober) runProbes(ctx context.Context) {
	// Get all pods and services
	pods, err := p.client.Clientset().CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list pods: %v\n", err)
		return
	}

	services, err := p.client.Clientset().CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list services: %v\n", err)
		return
	}

	// Probe service endpoints
	for _, svc := range services.Items {
		if svc.Spec.ClusterIP == "None" || svc.Spec.ClusterIP == "" {
			continue // Skip headless services for now
		}

		for _, port := range svc.Spec.Ports {
			// Find pods that can reach this service
			for _, pod := range pods.Items {
				if pod.Status.Phase != corev1.PodRunning {
					continue
				}

				// Perform probe
				result := p.probeTCP(pod, svc, port)
				p.addResult(result)
			}
		}
	}

	// Probe pod-to-pod connectivity (sample)
	p.probePodToPod(ctx, pods.Items)
}

// probeTCP performs a TCP connectivity probe
func (p *Prober) probeTCP(sourcePod corev1.Pod, targetSvc corev1.Service, port corev1.ServicePort) ProbeResult {
	result := ProbeResult{
		Timestamp:  time.Now(),
		SourcePod:  sourcePod.Name,
		SourceNS:   sourcePod.Namespace,
		TargetSvc:  targetSvc.Name,
		TargetNS:   targetSvc.Namespace,
		TargetIP:   targetSvc.Spec.ClusterIP,
		TargetPort: port.Port,
		ProbeType:  "TCP",
	}

	// Simulate TCP probe (in real implementation, this would be done from within the cluster)
	start := time.Now()
	
	address := fmt.Sprintf("%s:%d", targetSvc.Spec.ClusterIP, port.Port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	
	result.Latency = time.Since(start).Milliseconds()
	
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		conn.Close()
	}

	return result
}

// probeHTTP performs an HTTP connectivity probe
func (p *Prober) probeHTTP(sourcePod corev1.Pod, targetSvc corev1.Service, port corev1.ServicePort) ProbeResult {
	result := ProbeResult{
		Timestamp:  time.Now(),
		SourcePod:  sourcePod.Name,
		SourceNS:   sourcePod.Namespace,
		TargetSvc:  targetSvc.Name,
		TargetNS:   targetSvc.Namespace,
		TargetIP:   targetSvc.Spec.ClusterIP,
		TargetPort: port.Port,
		ProbeType:  "HTTP",
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	
	url := fmt.Sprintf("http://%s:%d/healthz", targetSvc.Spec.ClusterIP, port.Port)
	resp, err := client.Get(url)
	
	result.Latency = time.Since(start).Milliseconds()
	
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		result.StatusCode = resp.StatusCode
		resp.Body.Close()
	}

	return result
}

// probePodToPod performs pod-to-pod connectivity probes
func (p *Prober) probePodToPod(ctx context.Context, pods []corev1.Pod) {
	// Sample a subset of pods to avoid O(n²) complexity
	maxProbes := 10
	probeCount := 0

	for i, sourcePod := range pods {
		if sourcePod.Status.Phase != corev1.PodRunning {
			continue
		}

		for j, targetPod := range pods {
			if i == j || targetPod.Status.Phase != corev1.PodRunning {
				continue
			}

			if probeCount >= maxProbes {
				return
			}

			// Probe each container port
			for _, container := range targetPod.Spec.Containers {
				for _, port := range container.Ports {
					result := ProbeResult{
						Timestamp:  time.Now(),
						SourcePod:  sourcePod.Name,
						SourceNS:   sourcePod.Namespace,
						TargetPod:  targetPod.Name,
						TargetNS:   targetPod.Namespace,
						TargetIP:   targetPod.Status.PodIP,
						TargetPort: port.ContainerPort,
						ProbeType:  "TCP",
					}

					// Simulate probe
					start := time.Now()
					address := fmt.Sprintf("%s:%d", targetPod.Status.PodIP, port.ContainerPort)
					conn, err := net.DialTimeout("tcp", address, 2*time.Second)
					
					result.Latency = time.Since(start).Milliseconds()
					
					if err != nil {
						result.Success = false
						result.Error = err.Error()
					} else {
						result.Success = true
						conn.Close()
					}

					p.addResult(result)
					probeCount++
				}
			}
		}
	}
}

// addResult adds a probe result to the collection
func (p *Prober) addResult(result ProbeResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.results = append(p.results, result)
	
	// Keep only last 1000 results to avoid memory growth
	if len(p.results) > 1000 {
		p.results = p.results[len(p.results)-1000:]
	}
}

// GetResults returns all probe results
func (p *Prober) GetResults() []ProbeResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	results := make([]ProbeResult, len(p.results))
	copy(results, p.results)
	return results
}

// GetRecentResults returns probe results from the last duration
func (p *Prober) GetRecentResults(duration time.Duration) []ProbeResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var recent []ProbeResult

	for _, result := range p.results {
		if result.Timestamp.After(cutoff) {
			recent = append(recent, result)
		}
	}

	return recent
}

// GetFailedProbes returns only failed probe results
func (p *Prober) GetFailedProbes() []ProbeResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var failed []ProbeResult
	for _, result := range p.results {
		if !result.Success {
			failed = append(failed, result)
		}
	}

	return failed
}

// probeGRPC performs a gRPC health check probe
func (p *Prober) probeGRPC(sourcePod corev1.Pod, targetSvc corev1.Service, port corev1.ServicePort) ProbeResult {
	result := ProbeResult{
		Timestamp:  time.Now(),
		SourcePod:  sourcePod.Name,
		SourceNS:   sourcePod.Namespace,
		TargetSvc:  targetSvc.Name,
		TargetNS:   targetSvc.Namespace,
		TargetIP:   targetSvc.Spec.ClusterIP,
		TargetPort: port.Port,
		ProbeType:  "gRPC",
	}

	start := time.Now()
	
	address := fmt.Sprintf("%s:%d", targetSvc.Spec.ClusterIP, port.Port)
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Latency = time.Since(start).Milliseconds()
		return result
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	
	result.Latency = time.Since(start).Milliseconds()
	
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = resp.Status == healthpb.HealthCheckResponse_SERVING
		if !result.Success {
			result.Error = fmt.Sprintf("Service status: %s", resp.Status.String())
		}
	}

	return result
}

// detectPacketLoss performs multiple probes to detect packet loss
func (p *Prober) detectPacketLoss(sourcePod corev1.Pod, targetIP string, targetPort int32) float64 {
	probes := 10
	failed := 0
	
	for i := 0; i < probes; i++ {
		address := fmt.Sprintf("%s:%d", targetIP, targetPort)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		
		if err != nil {
			failed++
		} else {
			conn.Close()
		}
		
		// Small delay between probes
		time.Sleep(100 * time.Millisecond)
	}
	
	return float64(failed) / float64(probes) * 100
}

// CalculatePacketLoss calculates packet loss for recent probes
func (p *Prober) CalculatePacketLoss(targetIP string, duration time.Duration) float64 {
	recent := p.GetRecentResults(duration)
	
	if len(recent) == 0 {
		return 0
	}
	
	failed := 0
	total := 0
	
	for _, result := range recent {
		if result.TargetIP == targetIP {
			total++
			if !result.Success {
				failed++
			}
		}
	}
	
	if total == 0 {
		return 0
	}
	
	return float64(failed) / float64(total) * 100
}
