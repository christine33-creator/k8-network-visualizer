package flowcollector

import (
	"context"
	"fmt"
	"time"
)

// FlowCollectorInterface defines the common interface for all flow collectors
// This allows the system to work with different backends (Cilium, Istio, universal, etc.)
type FlowCollectorInterface interface {
	// Start begins collecting flows
	Start() error

	// Stop halts flow collection
	Stop()

	// GetFlows returns recent flows (limit = max number to return)
	GetFlows(limit int) []*Flow

	// GetFlowMetrics returns aggregated flow metrics by pod pairs
	GetFlowMetrics() map[string]*FlowMetric

	// GetStats returns collector statistics
	GetStats() map[string]interface{}
}

// Flow represents a network flow (common structure for all collectors)
type Flow struct {
	ID              string    `json:"id"`
	SourcePod       string    `json:"source_pod"`
	SourceIP        string    `json:"source_ip"`
	SourcePort      int       `json:"source_port"`
	SourceNamespace string    `json:"source_namespace"`
	DestPod         string    `json:"dest_pod"`
	DestIP          string    `json:"dest_ip"`
	DestPort        int       `json:"dest_port"`
	DestNamespace   string    `json:"dest_namespace"`
	Protocol        string    `json:"protocol"`
	FlowType        string    `json:"flow_type"`
	BytesSent       int64     `json:"bytes_sent"`
	PacketsSent     int64     `json:"packets_sent"`
	BytesPerSec     float64   `json:"bytes_per_sec"`
	PacketsPerSec   float64   `json:"packets_per_sec"`
	Direction       string    `json:"direction"`
	IsReply         bool      `json:"is_reply"`
	Verdict         string    `json:"verdict"`
	DropReason      string    `json:"drop_reason,omitempty"`
	L7Protocol      string    `json:"l7_protocol,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// FlowMetric represents aggregated flow metrics between pod pairs
type FlowMetric struct {
	SourcePod       string    `json:"source_pod"`
	SourceNamespace string    `json:"source_namespace"`
	DestPod         string    `json:"dest_pod"`
	DestNamespace   string    `json:"dest_namespace"`
	BytesPerSec     float64   `json:"bytes_per_sec"`
	PacketsPerSec   float64   `json:"packets_per_sec"`
	ConnectionCount int       `json:"connection_count"`
	ErrorRate       float64   `json:"error_rate"`
	Protocol        string    `json:"protocol"`
	LastSeen        time.Time `json:"last_seen"`
	IsActive        bool      `json:"is_active"`
	Direction       string    `json:"direction"`
}

// CollectorType represents the type of flow collector
type CollectorType string

const (
	CollectorTypeUniversal CollectorType = "universal" // Works everywhere (conntrack + iptables)
	CollectorTypeCilium    CollectorType = "cilium"    // Cilium Hubble (enhanced)
	CollectorTypeIstio     CollectorType = "istio"     // Istio/Envoy metrics (enhanced)
	CollectorTypeCalico    CollectorType = "calico"    // Calico Felix metrics (enhanced)
)

// CollectorFactory creates the appropriate flow collector based on environment
type CollectorFactory struct {
	ctx context.Context
}

// NewCollectorFactory creates a new collector factory
func NewCollectorFactory(ctx context.Context) *CollectorFactory {
	return &CollectorFactory{ctx: ctx}
}

// CreateCollector auto-detects and creates the best available flow collector
// Priority: Universal (always works) -> Enhanced (if available)
func (f *CollectorFactory) CreateCollector() (FlowCollectorInterface, CollectorType, error) {
	// Try to detect enhanced collectors first
	if collector, err := f.tryCreateCiliumCollector(); err == nil {
		return collector, CollectorTypeCilium, nil
	}

	if collector, err := f.tryCreateIstioCollector(); err == nil {
		return collector, CollectorTypeIstio, nil
	}

	if collector, err := f.tryCreateCalicoCollector(); err == nil {
		return collector, CollectorTypeCalico, nil
	}

	// Fall back to universal collector (ALWAYS works)
	collector := NewUniversalFlowCollector(UniversalFlowCollectorConfig{
		MaxRecentFlows: 10000,
		UpdateInterval: 5 * time.Second,
	})

	return collector, CollectorTypeUniversal, nil
}

// tryCreateCiliumCollector attempts to create a Cilium Hubble collector
func (f *CollectorFactory) tryCreateCiliumCollector() (FlowCollectorInterface, error) {
	// Check if Hubble is available
	// Try to connect to hubble-relay.kube-system.svc.cluster.local:80
	// If connection fails, return error
	
	// TODO: Implement Cilium detection
	// For now, return error to fall back to universal
	return nil, fmt.Errorf("Cilium Hubble not detected")
}

// tryCreateIstioCollector attempts to create an Istio metrics collector
func (f *CollectorFactory) tryCreateIstioCollector() (FlowCollectorInterface, error) {
	// Check if Istio is installed (look for istio-system namespace)
	// Check for Prometheus with Istio metrics
	
	// TODO: Implement Istio detection
	return nil, fmt.Errorf("Istio not detected")
}

// tryCreateCalicoCollector attempts to create a Calico metrics collector
func (f *CollectorFactory) tryCreateCalicoCollector() (FlowCollectorInterface, error) {
	// Check if Calico is installed (look for calico-system namespace)
	// Check for Felix Prometheus metrics
	
	// TODO: Implement Calico detection
	return nil, fmt.Errorf("Calico not detected")
}
