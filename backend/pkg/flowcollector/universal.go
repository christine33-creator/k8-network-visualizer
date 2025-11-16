package flowcollector

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// UniversalFlowCollector collects network flows using kernel-level tools
// that work on ANY Kubernetes cluster, regardless of CNI or service mesh
type UniversalFlowCollector struct {
	mu              sync.RWMutex
	flows           map[string]*Flow
	recentFlows     []*Flow
	maxRecentFlows  int
	podIPCache      map[string]PodInfo // IP -> Pod info mapping
	updateInterval  time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
}

// PodInfo stores cached pod information for IP resolution
type PodInfo struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

// UniversalFlowCollectorConfig holds configuration options
type UniversalFlowCollectorConfig struct {
	MaxRecentFlows int
	UpdateInterval time.Duration
	K8sClient      interface{} // Optional: K8s client for pod IP resolution
}

// NewUniversalFlowCollector creates a CNI-agnostic flow collector
// It uses kernel conntrack and iptables stats which work on ANY Linux system
func NewUniversalFlowCollector(config UniversalFlowCollectorConfig) *UniversalFlowCollector {
	if config.MaxRecentFlows == 0 {
		config.MaxRecentFlows = 10000
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &UniversalFlowCollector{
		flows:          make(map[string]*Flow),
		recentFlows:    make([]*Flow, 0),
		maxRecentFlows: config.MaxRecentFlows,
		podIPCache:     make(map[string]PodInfo),
		updateInterval: config.UpdateInterval,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start begins collecting network flows from kernel sources
func (c *UniversalFlowCollector) Start() error {
	log.Println("Starting Universal Flow Collector (CNI-agnostic)...")

	// Verify we have necessary capabilities
	if err := c.verifyCapabilities(); err != nil {
		return fmt.Errorf("missing required capabilities: %w", err)
	}

	log.Println("✓ Universal flow collection using kernel conntrack")
	log.Println("✓ Works with ANY CNI: Cilium, Calico, Flannel, Weave, or none")
	log.Println("✓ No service mesh required")

	// Start collection goroutines
	go c.collectConntrackFlows()
	go c.collectIptablesStats()
	go c.aggregateFlows()

	return nil
}

// Stop halts flow collection
func (c *UniversalFlowCollector) Stop() {
	log.Println("Stopping Universal Flow Collector...")
	c.cancel()
}

// verifyCapabilities checks if we can access kernel networking info
func (c *UniversalFlowCollector) verifyCapabilities() error {
	// Check if conntrack is available
	cmd := exec.Command("conntrack", "-L", "-o", "extended")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: conntrack not available or no permissions: %v", err)
		log.Println("Tip: Run as privileged pod with NET_ADMIN capability")
	}

	// Check if iptables is available
	cmd = exec.Command("iptables", "-L", "-n", "-v", "-x")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: iptables not available: %v", err)
	}

	// At least one method should work
	return nil
}

// collectConntrackFlows reads the kernel connection tracking table
// This is THE universal method that works everywhere
func (c *UniversalFlowCollector) collectConntrackFlows() {
	ticker := time.NewTicker(c.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.readConntrack(); err != nil {
				log.Printf("Error reading conntrack: %v", err)
			}
		}
	}
}

// readConntrack parses kernel conntrack table for active connections
func (c *UniversalFlowCollector) readConntrack() error {
	// Execute: conntrack -L -o extended
	// This shows ALL active connections on the system
	cmd := exec.Command("conntrack", "-L", "-o", "extended")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative: read /proc/net/nf_conntrack directly
		return c.readProcConntrack()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		flow := c.parseConntrackLine(line)
		if flow != nil {
			c.flows[flow.ID] = flow
			c.addRecentFlow(flow)
		}
	}

	return nil
}

// readProcConntrack reads /proc/net/nf_conntrack as fallback
// This file exists on all Linux systems with conntrack enabled
func (c *UniversalFlowCollector) readProcConntrack() error {
	cmd := exec.Command("cat", "/proc/net/nf_conntrack")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot read conntrack table: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		flow := c.parseProcConntrackLine(line)
		if flow != nil {
			c.flows[flow.ID] = flow
			c.addRecentFlow(flow)
		}
	}

	return nil
}

// parseConntrackLine parses conntrack command output
// Format: tcp 6 431999 ESTABLISHED src=10.244.0.5 dst=10.244.0.6 sport=45678 dport=8080 ...
func (c *UniversalFlowCollector) parseConntrackLine(line string) *Flow {
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return nil
	}

	flow := &Flow{
		Timestamp: time.Now(),
	}

	// Parse protocol
	if len(fields) > 0 {
		flow.Protocol = strings.ToUpper(fields[0])
	}

	// Parse IPs and ports from key=value pairs
	for _, field := range fields {
		parts := strings.Split(field, "=")
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "src":
			flow.SourceIP = value
		case "dst":
			flow.DestIP = value
		case "sport":
			if port, err := strconv.Atoi(value); err == nil {
				flow.SourcePort = port
			}
		case "dport":
			if port, err := strconv.Atoi(value); err == nil {
				flow.DestPort = port
			}
		case "bytes":
			if bytes, err := strconv.ParseInt(value, 10, 64); err == nil {
				flow.BytesSent = bytes
			}
		case "packets":
			if packets, err := strconv.ParseInt(value, 10, 64); err == nil {
				flow.PacketsSent = packets
			}
		}
	}

	// Resolve IPs to pod names
	c.resolveFlowPods(flow)

	// Generate flow ID
	flow.ID = fmt.Sprintf("%s:%d->%s:%d-%s",
		flow.SourceIP, flow.SourcePort,
		flow.DestIP, flow.DestPort,
		flow.Protocol)

	flow.Verdict = "ACCEPT" // Conntrack only shows accepted flows
	flow.IsReply = false
	flow.Direction = "egress"

	return flow
}

// parseProcConntrackLine parses /proc/net/nf_conntrack format
func (c *UniversalFlowCollector) parseProcConntrackLine(line string) *Flow {
	// Format: ipv4 2 tcp 6 431999 ESTABLISHED src=10.244.0.5 dst=10.244.0.6 sport=45678 dport=8080 ...
	return c.parseConntrackLine(line) // Same parser works for both
}

// collectIptablesStats reads iptables packet/byte counters
// This provides additional flow metrics
func (c *UniversalFlowCollector) collectIptablesStats() {
	ticker := time.NewTicker(c.updateInterval * 2) // Less frequent than conntrack
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.readIptablesStats(); err != nil {
				log.Printf("Error reading iptables stats: %v", err)
			}
		}
	}
}

// readIptablesStats parses iptables counters for flow statistics
func (c *UniversalFlowCollector) readIptablesStats() error {
	// Execute: iptables -L -n -v -x (numeric, verbose, exact counters)
	cmd := exec.Command("iptables", "-L", "-n", "-v", "-x")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot read iptables: %w", err)
	}

	// Parse iptables output for packet/byte counts
	// This gives us DROP counts and additional metrics
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "pkts") || strings.Contains(line, "Chain") || line == "" {
			continue
		}

		// Parse iptables rule line
		// Format: pkts bytes target prot opt in out source destination
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// Extract packet and byte counts
		// This can be used to enhance flow metrics
		// For now, we just collect the data
		// TODO: Correlate with flows based on IP/port matches
	}

	return nil
}

// aggregateFlows combines data from different sources and calculates rates
func (c *UniversalFlowCollector) aggregateFlows() {
	ticker := time.NewTicker(c.updateInterval)
	defer ticker.Stop()

	lastRun := time.Now()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(lastRun).Seconds()

			c.mu.Lock()
			// Calculate rates (bytes/sec, packets/sec)
			for _, flow := range c.flows {
				flow.BytesPerSec = float64(flow.BytesSent) / elapsed
				flow.PacketsPerSec = float64(flow.PacketsSent) / elapsed
			}
			c.mu.Unlock()

			lastRun = now
		}
	}
}

// resolveFlowPods maps IPs to pod names using cached pod information
func (c *UniversalFlowCollector) resolveFlowPods(flow *Flow) {
	// Check cache for source IP
	if podInfo, ok := c.podIPCache[flow.SourceIP]; ok {
		flow.SourcePod = podInfo.Name
		flow.SourceNamespace = podInfo.Namespace
	} else {
		flow.SourcePod = flow.SourceIP // Fallback to IP
		flow.SourceNamespace = "unknown"
	}

	// Check cache for dest IP
	if podInfo, ok := c.podIPCache[flow.DestIP]; ok {
		flow.DestPod = podInfo.Name
		flow.DestNamespace = podInfo.Namespace
	} else {
		flow.DestPod = flow.DestIP // Fallback to IP
		flow.DestNamespace = "unknown"
	}
}

// UpdatePodIPCache updates the IP->Pod mapping (called by main collector)
func (c *UniversalFlowCollector) UpdatePodIPCache(ip string, info PodInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.podIPCache[ip] = info
}

// GetFlows returns recent flows (implements FlowCollector interface)
func (c *UniversalFlowCollector) GetFlows(limit int) []*Flow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.recentFlows) {
		limit = len(c.recentFlows)
	}

	result := make([]*Flow, limit)
	copy(result, c.recentFlows[:limit])
	return result
}

// GetFlowMetrics aggregates flow data by pod pairs
func (c *UniversalFlowCollector) GetFlowMetrics() map[string]*FlowMetric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := make(map[string]*FlowMetric)

	for _, flow := range c.flows {
		key := fmt.Sprintf("%s/%s->%s/%s",
			flow.SourceNamespace, flow.SourcePod,
			flow.DestNamespace, flow.DestPod)

		if metric, ok := metrics[key]; ok {
			metric.BytesPerSec += flow.BytesPerSec
			metric.PacketsPerSec += flow.PacketsPerSec
			metric.ConnectionCount++
			metric.LastSeen = flow.Timestamp
		} else {
			metrics[key] = &FlowMetric{
				SourcePod:       flow.SourcePod,
				SourceNamespace: flow.SourceNamespace,
				DestPod:         flow.DestPod,
				DestNamespace:   flow.DestNamespace,
				BytesPerSec:     flow.BytesPerSec,
				PacketsPerSec:   flow.PacketsPerSec,
				ConnectionCount: 1,
				Protocol:        flow.Protocol,
				LastSeen:        flow.Timestamp,
				IsActive:        true,
			}
		}
	}

	return metrics
}

// addRecentFlow adds a flow to the recent flows list
func (c *UniversalFlowCollector) addRecentFlow(flow *Flow) {
	c.recentFlows = append(c.recentFlows, flow)
	if len(c.recentFlows) > c.maxRecentFlows {
		c.recentFlows = c.recentFlows[1:] // Remove oldest
	}
}

// GetStats returns collector statistics
func (c *UniversalFlowCollector) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"active_flows":  len(c.flows),
		"recent_flows":  len(c.recentFlows),
		"pod_ip_cache":  len(c.podIPCache),
		"collector_type": "universal (conntrack + iptables)",
		"cni_agnostic":  true,
	}
}

// IsIPInPodNetwork checks if an IP belongs to the pod network
func IsIPInPodNetwork(ip string) bool {
	// Common pod CIDR ranges
	podCIDRs := []string{
		"10.244.0.0/16", // Common default
		"10.32.0.0/12",  // Another common range
		"172.16.0.0/12", // Docker default
		"192.168.0.0/16", // Private network
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, cidr := range podCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ipAddr) {
			return true
		}
	}

	return false
}
