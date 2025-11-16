package flowcollector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	observer "github.com/cilium/cilium/api/v1/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// FlowType represents the type of network flow
type FlowType string

const (
	FlowTypeL3L4     FlowType = "l3_l4"      // Network layer (IP) and transport layer (TCP/UDP)
	FlowTypeL7       FlowType = "l7"         // Application layer (HTTP, gRPC, DNS)
	FlowTypeDrop     FlowType = "drop"       // Dropped packets
	FlowTypePolicyDeny FlowType = "policy_deny" // Denied by network policy
)

// Protocol represents network protocols
type Protocol string

const (
	ProtocolTCP    Protocol = "TCP"
	ProtocolUDP    Protocol = "UDP"
	ProtocolICMP   Protocol = "ICMP"
	ProtocolHTTP   Protocol = "HTTP"
	ProtocolHTTPS  Protocol = "HTTPS"
	ProtocolGRPC   Protocol = "gRPC"
	ProtocolDNS    Protocol = "DNS"
	ProtocolUnknown Protocol = "UNKNOWN"
)

// NetworkFlow represents a captured network flow
type NetworkFlow struct {
	ID               string            `json:"id"`
	SourcePod        string            `json:"source_pod"`
	SourceIP         string            `json:"source_ip"`
	SourcePort       uint32            `json:"source_port"`
	SourceNamespace  string            `json:"source_namespace"`
	DestPod          string            `json:"dest_pod"`
	DestIP           string            `json:"dest_ip"`
	DestPort         uint32            `json:"dest_port"`
	DestNamespace    string            `json:"dest_namespace"`
	Protocol         Protocol          `json:"protocol"`
	FlowType         FlowType          `json:"flow_type"`
	BytesSent        uint64            `json:"bytes_sent"`
	PacketsSent      uint64            `json:"packets_sent"`
	Direction        string            `json:"direction"` // ingress, egress
	IsReply          bool              `json:"is_reply"`
	Verdict          string            `json:"verdict"` // FORWARDED, DROPPED, ERROR
	DropReason       string            `json:"drop_reason,omitempty"`
	L7Protocol       string            `json:"l7_protocol,omitempty"`
	L7Details        map[string]string `json:"l7_details,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
	TraceObservation string            `json:"trace_observation,omitempty"`
}

// FlowMetrics represents aggregated flow metrics
type FlowMetrics struct {
	SourceID        string    `json:"source_id"`
	DestID          string    `json:"dest_id"`
	BytesPerSec     float64   `json:"bytes_per_sec"`
	PacketsPerSec   float64   `json:"packets_per_sec"`
	ConnectionCount int64     `json:"connection_count"`
	ErrorRate       float64   `json:"error_rate"`
	AvgLatency      float64   `json:"avg_latency_ms,omitempty"`
	Protocol        Protocol  `json:"protocol"`
	LastSeen        time.Time `json:"last_seen"`
}

// FlowCollector collects network flows from Cilium Hubble or eBPF
type FlowCollector struct {
	hubbleAddr   string
	conn         *grpc.ClientConn
	client       observer.ObserverClient
	
	mu           sync.RWMutex
	flows        []NetworkFlow
	flowMetrics  map[string]*FlowMetrics // key: source_id->dest_id
	
	maxFlows     int
	metricWindow time.Duration
	
	// Callbacks for real-time updates
	onFlowCallback func(flow NetworkFlow)
}

// NewFlowCollector creates a new flow collector
func NewFlowCollector(hubbleAddr string) *FlowCollector {
	return &FlowCollector{
		hubbleAddr:   hubbleAddr,
		flows:        make([]NetworkFlow, 0),
		flowMetrics:  make(map[string]*FlowMetrics),
		maxFlows:     10000, // Keep last 10k flows
		metricWindow: 60 * time.Second,
	}
}

// SetOnFlowCallback sets a callback for real-time flow notifications
func (fc *FlowCollector) SetOnFlowCallback(callback func(flow NetworkFlow)) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.onFlowCallback = callback
}

// Connect establishes connection to Hubble relay
func (fc *FlowCollector) Connect(ctx context.Context) error {
	log.Printf("Connecting to Hubble at %s...", fc.hubbleAddr)
	
	conn, err := grpc.DialContext(
		ctx,
		fc.hubbleAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Hubble: %w", err)
	}
	
	fc.conn = conn
	fc.client = observer.NewObserverClient(conn)
	
	log.Printf("Successfully connected to Hubble")
	return nil
}

// Start begins collecting flows
func (fc *FlowCollector) Start(ctx context.Context) error {
	if fc.client == nil {
		return fmt.Errorf("not connected to Hubble - call Connect() first")
	}
	
	// Create flow request - watch all flows
	req := &observer.GetFlowsRequest{
		Follow: true, // Stream flows in real-time
		Whitelist: []*observer.FlowFilter{
			{}, // Empty filter = all flows
		},
	}
	
	log.Println("Starting flow collection...")
	stream, err := fc.client.GetFlows(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get flows: %w", err)
	}
	
	// Start metric aggregation goroutine
	go fc.aggregateMetrics(ctx)
	
	// Process flows
	for {
		select {
		case <-ctx.Done():
			log.Println("Flow collection stopped")
			return ctx.Err()
		default:
		}
		
		response, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving flow: %v", err)
			return err
		}
		
		flow := fc.parseHubbleFlow(response)
		if flow != nil {
			fc.addFlow(*flow)
			
			// Trigger callback if set
			fc.mu.RLock()
			callback := fc.onFlowCallback
			fc.mu.RUnlock()
			
			if callback != nil {
				go callback(*flow)
			}
		}
	}
}

// parseHubbleFlow converts Hubble flow to our NetworkFlow structure
func (fc *FlowCollector) parseHubbleFlow(response *observer.GetFlowsResponse) *NetworkFlow {
	hubbleFlow := response.GetFlow()
	if hubbleFlow == nil {
		return nil
	}
	
	flow := &NetworkFlow{
		ID:        fmt.Sprintf("%d", hubbleFlow.GetTime().GetSeconds()),
		Timestamp: time.Unix(hubbleFlow.GetTime().GetSeconds(), int64(hubbleFlow.GetTime().GetNanos())),
		Verdict:   hubbleFlow.GetVerdict().String(),
	}
	
	// IP addresses
	if ip := hubbleFlow.GetIP(); ip != nil {
		flow.SourceIP = ip.GetSource()
		flow.DestIP = ip.GetDestination()
	}
	
	// Source endpoint information
	if src := hubbleFlow.GetSource(); src != nil {
		flow.SourceNamespace = src.GetNamespace()
		if len(src.GetPodName()) > 0 {
			flow.SourcePod = fmt.Sprintf("%s/%s", src.GetNamespace(), src.GetPodName())
		}
	}
	
	// Destination endpoint information
	if dst := hubbleFlow.GetDestination(); dst != nil {
		flow.DestNamespace = dst.GetNamespace()
		if len(dst.GetPodName()) > 0 {
			flow.DestPod = fmt.Sprintf("%s/%s", dst.GetNamespace(), dst.GetPodName())
		}
	}
	
	// L4 protocol and ports
	if l4 := hubbleFlow.GetL4(); l4 != nil {
		if tcp := l4.GetTCP(); tcp != nil {
			flow.Protocol = ProtocolTCP
			flow.SourcePort = tcp.GetSourcePort()
			flow.DestPort = tcp.GetDestinationPort()
		} else if udp := l4.GetUDP(); udp != nil {
			flow.Protocol = ProtocolUDP
			flow.SourcePort = udp.GetSourcePort()
			flow.DestPort = udp.GetDestinationPort()
		} else if icmp := l4.GetICMPv4(); icmp != nil {
			flow.Protocol = ProtocolICMP
		}
	}
	
	// L7 protocol
	if l7 := hubbleFlow.GetL7(); l7 != nil {
		flow.FlowType = FlowTypeL7
		flow.L7Details = make(map[string]string)
		
		if http := l7.GetHttp(); http != nil {
			if flow.DestPort == 443 {
				flow.L7Protocol = string(ProtocolHTTPS)
			} else {
				flow.L7Protocol = string(ProtocolHTTP)
			}
			flow.L7Details["method"] = http.GetMethod()
			flow.L7Details["url"] = http.GetUrl()
			flow.L7Details["code"] = fmt.Sprintf("%d", http.GetCode())
		} else if dns := l7.GetDns(); dns != nil {
			flow.L7Protocol = string(ProtocolDNS)
			flow.L7Details["query"] = dns.GetQuery()
		}
	} else if hubbleFlow.GetIsReply() != nil {
		flow.FlowType = FlowTypeL3L4
	}
	
	// Flow direction
	flow.IsReply = hubbleFlow.GetIsReply() != nil && hubbleFlow.GetIsReply().GetValue()
	if hubbleFlow.GetTrafficDirection().String() == "INGRESS" {
		flow.Direction = "ingress"
	} else {
		flow.Direction = "egress"
	}
	
	// Drop information
	if hubbleFlow.GetDropReason() != 0 {
		flow.FlowType = FlowTypeDrop
		flow.DropReason = hubbleFlow.GetDropReasonDesc().String()
		if flow.DropReason == "" {
			flow.DropReason = fmt.Sprintf("Reason code: %d", hubbleFlow.GetDropReason())
		}
	}
	
	// Packet/byte counts (if available in summary)
	if summary := hubbleFlow.GetSummary(); summary != "" {
		flow.TraceObservation = summary
	}
	
	// Default packet count
	flow.PacketsSent = 1
	flow.BytesSent = 0 // Hubble doesn't always provide byte count per flow
	
	return flow
}

// addFlow adds a flow to the collection
func (fc *FlowCollector) addFlow(flow NetworkFlow) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	// Add to flow list
	fc.flows = append(fc.flows, flow)
	
	// Trim if exceeds max
	if len(fc.flows) > fc.maxFlows {
		fc.flows = fc.flows[len(fc.flows)-fc.maxFlows:]
	}
	
	// Update metrics
	fc.updateMetrics(flow)
}

// updateMetrics updates aggregated metrics for a flow
func (fc *FlowCollector) updateMetrics(flow NetworkFlow) {
	// Skip if source or dest is empty
	if flow.SourcePod == "" || flow.DestPod == "" {
		return
	}
	
	key := fmt.Sprintf("%s->%s", flow.SourcePod, flow.DestPod)
	
	metric, exists := fc.flowMetrics[key]
	if !exists {
		metric = &FlowMetrics{
			SourceID:        flow.SourcePod,
			DestID:          flow.DestPod,
			Protocol:        flow.Protocol,
			ConnectionCount: 0,
			LastSeen:        flow.Timestamp,
		}
		fc.flowMetrics[key] = metric
	}
	
	metric.ConnectionCount++
	metric.LastSeen = flow.Timestamp
	
	// Update error rate
	if flow.Verdict == "DROPPED" || flow.Verdict == "ERROR" {
		metric.ErrorRate = (metric.ErrorRate*float64(metric.ConnectionCount-1) + 1.0) / float64(metric.ConnectionCount)
	} else {
		metric.ErrorRate = metric.ErrorRate * float64(metric.ConnectionCount-1) / float64(metric.ConnectionCount)
	}
}

// aggregateMetrics periodically calculates bytes/sec and packets/sec
func (fc *FlowCollector) aggregateMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fc.calculateRates()
		}
	}
}

// calculateRates calculates throughput rates
func (fc *FlowCollector) calculateRates() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	now := time.Now()
	windowStart := now.Add(-fc.metricWindow)
	
	// Reset metrics
	for key := range fc.flowMetrics {
		fc.flowMetrics[key].BytesPerSec = 0
		fc.flowMetrics[key].PacketsPerSec = 0
	}
	
	// Aggregate flows within window
	for _, flow := range fc.flows {
		if flow.Timestamp.Before(windowStart) {
			continue
		}
		
		key := fmt.Sprintf("%s->%s", flow.SourcePod, flow.DestPod)
		
		if metric, exists := fc.flowMetrics[key]; exists {
			metric.BytesPerSec += float64(flow.BytesSent)
			metric.PacketsPerSec += float64(flow.PacketsSent)
		}
	}
	
	// Normalize by window duration
	windowSeconds := fc.metricWindow.Seconds()
	for _, metric := range fc.flowMetrics {
		metric.BytesPerSec /= windowSeconds
		metric.PacketsPerSec /= windowSeconds
	}
}

// GetFlows returns recent flows
func (fc *FlowCollector) GetFlows(limit int) []NetworkFlow {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	
	if limit <= 0 || limit > len(fc.flows) {
		limit = len(fc.flows)
	}
	
	// Return most recent flows
	start := len(fc.flows) - limit
	flows := make([]NetworkFlow, limit)
	copy(flows, fc.flows[start:])
	
	return flows
}

// GetFlowMetrics returns aggregated flow metrics
func (fc *FlowCollector) GetFlowMetrics() []FlowMetrics {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	
	metrics := make([]FlowMetrics, 0, len(fc.flowMetrics))
	for _, metric := range fc.flowMetrics {
		metrics = append(metrics, *metric)
	}
	
	return metrics
}

// GetFlowsByPod returns flows for a specific pod
func (fc *FlowCollector) GetFlowsByPod(podID string) []NetworkFlow {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	
	flows := make([]NetworkFlow, 0)
	for _, flow := range fc.flows {
		if flow.SourcePod == podID || flow.DestPod == podID {
			flows = append(flows, flow)
		}
	}
	
	return flows
}

// Close closes the Hubble connection
func (fc *FlowCollector) Close() error {
	if fc.conn != nil {
		return fc.conn.Close()
	}
	return nil
}
