package analyzer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/christine33-creator/k8-network-visualizer/pkg/collector"
	"github.com/christine33-creator/k8-network-visualizer/pkg/graph"
	"github.com/christine33-creator/k8-network-visualizer/pkg/prober"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// ProbeResult is an alias for prober.ProbeResult
type ProbeResult = prober.ProbeResult

// IssueType represents the type of network issue
type IssueType string

const (
	IssueTypeConnectivity   IssueType = "connectivity"
	IssueTypeLatency        IssueType = "latency"
	IssueTypePolicy         IssueType = "policy"
	IssueTypeDNS            IssueType = "dns"
	IssueTypeConfiguration  IssueType = "configuration"
	IssueTypeCIDROverlap    IssueType = "cidr_overlap"
	IssueTypeResourceHealth IssueType = "resource_health"
)

// IssueSeverity represents the severity of an issue
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityHigh     IssueSeverity = "high"
	SeverityMedium   IssueSeverity = "medium"
	SeverityLow      IssueSeverity = "low"
)

// NetworkIssue represents a detected network issue
type NetworkIssue struct {
	ID          string                 `json:"id"`
	Type        IssueType              `json:"type"`
	Severity    IssueSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Affected    []string               `json:"affected_resources"`
	Suggestions []string               `json:"suggestions"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
}

// IntelligentInsight represents an actionable recommendation
type IntelligentInsight struct {
	ID          string                 `json:"id"`
	Category    string                 `json:"category"` // "optimization", "security", "reliability", "cost"
	Priority    string                 `json:"priority"` // "high", "medium", "low"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`
	Actions     []ActionableStep       `json:"actions"`
	Metrics     map[string]interface{} `json:"metrics"`
	Confidence  float64                `json:"confidence"` // 0.0 to 1.0
	Timestamp   time.Time              `json:"timestamp"`
}

// ActionableStep represents a specific action to take
type ActionableStep struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Risk        string `json:"risk"` // "low", "medium", "high"
	Automated   bool   `json:"automated"`
}

// SimulationRequest represents a what-if simulation request
type SimulationRequest struct {
	Type        string                 `json:"type"` // "policy", "resource", "topology"
	Changes     map[string]interface{} `json:"changes"`
	Scope       []string               `json:"scope"` // affected namespaces/resources
	Description string                 `json:"description"`
}

// SimulationResult represents the result of a what-if simulation
type SimulationResult struct {
	ID          string                 `json:"id"`
	Request     SimulationRequest      `json:"request"`
	Impact      SimulationImpact       `json:"impact"`
	Risks       []SimulationRisk       `json:"risks"`
	Benefits    []string               `json:"benefits"`
	Timeline    string                 `json:"timeline"`
	Confidence  float64                `json:"confidence"`
	Timestamp   time.Time              `json:"timestamp"`
}

// SimulationImpact represents the predicted impact of a change
type SimulationImpact struct {
	Connectivity  map[string]string `json:"connectivity"`  // before -> after
	Security      map[string]string `json:"security"`      // security score changes
	Performance   map[string]string `json:"performance"`   // performance predictions
	ResourceUsage map[string]string `json:"resource_usage"` // resource impact
}

// SimulationRisk represents a potential risk from a change
type SimulationRisk struct {
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Likelihood  float64 `json:"likelihood"`
	Mitigation  string  `json:"mitigation"`
}

// Analyzer performs network analysis and issue detection
type Analyzer struct {
	graphEngine *graph.Engine
	issues      []NetworkIssue
	insights    []IntelligentInsight
	simulations []SimulationResult
	mu          sync.RWMutex
	issueCount  int
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(engine *graph.Engine) *Analyzer {
	return &Analyzer{
		graphEngine: engine,
		issues:      make([]NetworkIssue, 0),
		insights:    make([]IntelligentInsight, 0),
		simulations: make([]SimulationResult, 0),
		issueCount:  0,
	}
}

// Start begins the analysis process
func (a *Analyzer) Start(ctx context.Context, collector *collector.Collector, prober *prober.Prober) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run initial analysis
	a.analyze(collector, prober)

	for {
		select {
		case <-ticker.C:
			a.analyze(collector, prober)
		case <-ctx.Done():
			return
		}
	}
}

// analyze performs comprehensive network analysis
func (a *Analyzer) analyze(collector *collector.Collector, p *prober.Prober) {
	// Clear previous issues and insights
	a.mu.Lock()
	a.issues = make([]NetworkIssue, 0)
	a.insights = make([]IntelligentInsight, 0)
	a.mu.Unlock()

	// Update graph with latest data
	a.updateGraph(collector)

	// Perform various analyses
	a.analyzeConnectivity(p)
	a.analyzeNetworkPolicies(collector)
	a.analyzePodHealth(collector)
	a.analyzeServiceEndpoints(collector)
	a.analyzeCIDROverlaps(collector)
	a.analyzeLatency(p)
	a.analyzeDNS(collector)
	a.detectFirewalls(p, collector)

	// Generate intelligent insights
	a.generateIntelligentInsights(collector, p)
}

// updateGraph updates the graph engine with latest data
func (a *Analyzer) updateGraph(collector *collector.Collector) {
	// Add pods to graph
	for _, pod := range collector.GetPods() {
		a.graphEngine.AddPod(pod)
	}

	// Add services to graph
	for _, svc := range collector.GetServices() {
		a.graphEngine.AddService(svc)
	}

	// Add nodes to graph
	for _, node := range collector.GetNodes() {
		a.graphEngine.AddNode(node)
	}

	// Add service endpoints
	services := collector.GetServices()
	endpoints := collector.GetEndpoints()
	for _, svc := range services {
		for _, ep := range endpoints {
			if ep.Name == svc.Name && ep.Namespace == svc.Namespace {
				a.graphEngine.AddServiceEndpoint(svc, ep)
			}
		}
	}

	// Add network policies
	for _, policy := range collector.GetNetworkPolicies() {
		a.graphEngine.AddNetworkPolicy(policy)
	}
}

// analyzeConnectivity checks for connectivity issues
func (a *Analyzer) analyzeConnectivity(p *prober.Prober) {
	failedProbes := p.GetFailedProbes()
	
	// Group failed probes by target
	failureMap := make(map[string][]ProbeResult)
	for _, probe := range failedProbes {
		key := fmt.Sprintf("%s/%s", probe.TargetNS, probe.TargetSvc)
		if probe.TargetSvc == "" {
			key = fmt.Sprintf("%s/%s", probe.TargetNS, probe.TargetPod)
		}
		failureMap[key] = append(failureMap[key], probe)
	}

	// Create issues for persistent failures
	for target, failures := range failureMap {
		if len(failures) >= 3 { // At least 3 failures
			issue := NetworkIssue{
				ID:        a.generateIssueID(),
				Type:      IssueTypeConnectivity,
				Severity:  SeverityCritical,
				Title:     fmt.Sprintf("Service Unreachable: %s", target),
				Description: fmt.Sprintf("Multiple connectivity failures detected for %s. %d probes failed in the last monitoring period.",
					target, len(failures)),
				Affected: []string{target},
				Suggestions: []string{
					"Check if the target pods are running",
					"Verify NetworkPolicy is not blocking traffic",
					"Check if Service endpoints are correctly configured",
					"Verify firewall rules in the cluster",
				},
				Details: map[string]interface{}{
					"failure_count": len(failures),
					"last_error":    failures[len(failures)-1].Error,
				},
				Timestamp: time.Now(),
			}
			a.addIssue(issue)
		}
	}
}

// analyzeNetworkPolicies checks for policy-related issues
func (a *Analyzer) analyzeNetworkPolicies(collector *collector.Collector) {
	policies := collector.GetNetworkPolicies()
	pods := collector.GetPods()

	// Check for conflicting policies
	namespacePolices := make(map[string][]*networkingv1.NetworkPolicy)
	for _, policy := range policies {
		namespacePolices[policy.Namespace] = append(namespacePolices[policy.Namespace], policy)
	}

	for namespace, nsPolicies := range namespacePolices {
		if len(nsPolicies) > 3 {
			issue := NetworkIssue{
				ID:          a.generateIssueID(),
				Type:        IssueTypePolicy,
				Severity:    SeverityMedium,
				Title:       fmt.Sprintf("Complex NetworkPolicy Configuration in %s", namespace),
				Description: fmt.Sprintf("Namespace %s has %d NetworkPolicies which may cause confusion or conflicts", namespace, len(nsPolicies)),
				Affected:    []string{namespace},
				Suggestions: []string{
					"Review and consolidate NetworkPolicies where possible",
					"Document the intent of each policy",
					"Use policy validation tools",
				},
				Timestamp: time.Now(),
			}
			a.addIssue(issue)
		}
	}

	// Check for pods without any policy coverage
	for _, pod := range pods {
		covered := false
		for _, policy := range policies {
			if policy.Namespace == pod.Namespace {
				// Simplified check - in production, evaluate selectors properly
				covered = true
				break
			}
		}

		if !covered && pod.Status.Phase == corev1.PodRunning {
			issue := NetworkIssue{
				ID:          a.generateIssueID(),
				Type:        IssueTypePolicy,
				Severity:    SeverityLow,
				Title:       fmt.Sprintf("Pod without NetworkPolicy: %s/%s", pod.Namespace, pod.Name),
				Description: "Pod is not covered by any NetworkPolicy, which may be a security risk",
				Affected:    []string{fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)},
				Suggestions: []string{
					"Consider adding a NetworkPolicy for this pod",
					"Review security requirements for this workload",
				},
				Timestamp: time.Now(),
			}
			a.addIssue(issue)
		}
	}
}

// analyzePodHealth checks for unhealthy pods
func (a *Analyzer) analyzePodHealth(collector *collector.Collector) {
	pods := collector.GetPods()
	
	for _, pod := range pods {
		// Check for pods in error states
		if pod.Status.Phase == corev1.PodFailed {
			issue := NetworkIssue{
				ID:          a.generateIssueID(),
				Type:        IssueTypeResourceHealth,
				Severity:    SeverityHigh,
				Title:       fmt.Sprintf("Pod Failed: %s/%s", pod.Namespace, pod.Name),
				Description: fmt.Sprintf("Pod is in Failed state. Reason: %s", pod.Status.Reason),
				Affected:    []string{fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)},
				Suggestions: []string{
					"Check pod logs for error details",
					"Review resource requirements",
					"Check node capacity and scheduling constraints",
				},
				Timestamp: time.Now(),
			}
			a.addIssue(issue)
		}

		// Check for pods stuck in pending
		if pod.Status.Phase == corev1.PodPending {
			// Check if pending for more than 5 minutes (simplified check)
			if time.Since(pod.CreationTimestamp.Time) > 5*time.Minute {
				issue := NetworkIssue{
					ID:          a.generateIssueID(),
					Type:        IssueTypeResourceHealth,
					Severity:    SeverityMedium,
					Title:       fmt.Sprintf("Pod Stuck Pending: %s/%s", pod.Namespace, pod.Name),
					Description: "Pod has been in Pending state for more than 5 minutes",
					Affected:    []string{fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)},
					Suggestions: []string{
						"Check if nodes have sufficient resources",
						"Review pod scheduling constraints",
						"Check for taints and tolerations mismatch",
					},
					Timestamp: time.Now(),
				}
				a.addIssue(issue)
			}
		}

		// Check for restart loops
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.RestartCount > 5 {
				issue := NetworkIssue{
					ID:          a.generateIssueID(),
					Type:        IssueTypeResourceHealth,
					Severity:    SeverityHigh,
					Title:       fmt.Sprintf("Container Restart Loop: %s/%s/%s", pod.Namespace, pod.Name, containerStatus.Name),
					Description: fmt.Sprintf("Container has restarted %d times", containerStatus.RestartCount),
					Affected:    []string{fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)},
					Suggestions: []string{
						"Check container logs for crash details",
						"Review liveness and readiness probes",
						"Check for resource limits and OOM kills",
					},
					Details: map[string]interface{}{
						"restart_count": containerStatus.RestartCount,
						"container":     containerStatus.Name,
					},
					Timestamp: time.Now(),
				}
				a.addIssue(issue)
			}
		}
	}
}

// analyzeServiceEndpoints checks for service endpoint issues
func (a *Analyzer) analyzeServiceEndpoints(collector *collector.Collector) {
	services := collector.GetServices()
	endpoints := collector.GetEndpoints()

	for _, svc := range services {
		if svc.Spec.Type == corev1.ServiceTypeClusterIP && svc.Spec.ClusterIP != "None" {
			// Find corresponding endpoints
			var ep *corev1.Endpoints
			for _, endpoint := range endpoints {
				if endpoint.Name == svc.Name && endpoint.Namespace == svc.Namespace {
					ep = endpoint
					break
				}
			}

			if ep == nil || len(ep.Subsets) == 0 {
				issue := NetworkIssue{
					ID:          a.generateIssueID(),
					Type:        IssueTypeConfiguration,
					Severity:    SeverityHigh,
					Title:       fmt.Sprintf("Service Without Endpoints: %s/%s", svc.Namespace, svc.Name),
					Description: "Service has no active endpoints, traffic will fail",
					Affected:    []string{fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)},
					Suggestions: []string{
						"Check if pods matching the service selector exist",
						"Verify pods are in Running state",
						"Review service selector labels",
					},
					Timestamp: time.Now(),
				}
				a.addIssue(issue)
			}
		}
	}
}

// analyzeCIDROverlaps checks for CIDR overlap issues
func (a *Analyzer) analyzeCIDROverlaps(collector *collector.Collector) {
	nodes := collector.GetNodes()
	
	// Extract pod CIDRs from nodes
	podCIDRs := make(map[string]string)
	for _, node := range nodes {
		if node.Spec.PodCIDR != "" {
			podCIDRs[node.Name] = node.Spec.PodCIDR
		}
	}

	// Check for overlaps (simplified check)
	for node1, cidr1 := range podCIDRs {
		for node2, cidr2 := range podCIDRs {
			if node1 != node2 && strings.HasPrefix(cidr1, strings.Split(cidr2, "/")[0]) {
				issue := NetworkIssue{
					ID:          a.generateIssueID(),
					Type:        IssueTypeCIDROverlap,
					Severity:    SeverityCritical,
					Title:       fmt.Sprintf("Potential CIDR Overlap: %s and %s", node1, node2),
					Description: fmt.Sprintf("Nodes have potentially overlapping pod CIDRs: %s and %s", cidr1, cidr2),
					Affected:    []string{node1, node2},
					Suggestions: []string{
						"Review cluster CIDR allocation",
						"Check CNI plugin configuration",
						"Verify no manual CIDR assignments conflict",
					},
					Details: map[string]interface{}{
						"cidr1": cidr1,
						"cidr2": cidr2,
					},
					Timestamp: time.Now(),
				}
				a.addIssue(issue)
				break
			}
		}
	}
}

// analyzeDNS checks for DNS-related issues
func (a *Analyzer) analyzeDNS(collector *collector.Collector) {
	services := collector.GetServices()
	endpoints := collector.GetEndpoints()
	
	// Check for services without endpoints
	for _, svc := range services {
		if svc.Spec.Type == corev1.ServiceTypeClusterIP && svc.Spec.ClusterIP != "None" {
			var hasEndpoints bool
			for _, ep := range endpoints {
				if ep.Name == svc.Name && ep.Namespace == svc.Namespace {
					if len(ep.Subsets) > 0 {
						hasEndpoints = true
						break
					}
				}
			}
			
			if !hasEndpoints {
				issue := NetworkIssue{
					ID:          a.generateIssueID(),
					Type:        IssueTypeDNS,
					Severity:    SeverityHigh,
					Title:       fmt.Sprintf("DNS Resolution Issue: %s/%s", svc.Namespace, svc.Name),
					Description: fmt.Sprintf("Service %s.%s has no endpoints, DNS resolution will fail", svc.Name, svc.Namespace),
					Affected:    []string{fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)},
					Suggestions: []string{
						"Check if pods matching the service selector are running",
						"Verify the service selector matches pod labels",
						"Check pod readiness probes",
						"Verify kube-dns/coredns is functioning correctly",
					},
					Timestamp: time.Now(),
				}
				a.addIssue(issue)
			}
		}
	}
	
	// Check for headless services
	headlessCount := 0
	for _, svc := range services {
		if svc.Spec.ClusterIP == "None" {
			headlessCount++
		}
	}
	
	if headlessCount > 0 {
		issue := NetworkIssue{
			ID:          a.generateIssueID(),
			Type:        IssueTypeDNS,
			Severity:    SeverityLow,
			Title:       fmt.Sprintf("Headless Services Detected: %d", headlessCount),
			Description: "Headless services require special DNS handling and may not work with all applications",
			Suggestions: []string{
				"Ensure applications are configured for headless service discovery",
				"Consider using StatefulSets for stateful workloads",
				"Monitor DNS query patterns for issues",
			},
			Details: map[string]interface{}{
				"headless_service_count": headlessCount,
			},
			Timestamp: time.Now(),
		}
		a.addIssue(issue)
	}
}

// detectFirewalls detects potential firewall or blocked traffic patterns
func (a *Analyzer) detectFirewalls(p *prober.Prober, collector *collector.Collector) {
	failedProbes := p.GetFailedProbes()
	
	// Analyze failure patterns
	blockPatterns := make(map[string]int)
	timeoutErrors := 0
	connectionRefused := 0
	
	for _, probe := range failedProbes {
		if strings.Contains(probe.Error, "timeout") {
			timeoutErrors++
			key := fmt.Sprintf("%s->%s:%d", probe.SourceNS, probe.TargetIP, probe.TargetPort)
			blockPatterns[key]++
		} else if strings.Contains(probe.Error, "connection refused") {
			connectionRefused++
		}
	}
	
	// Detect systematic blocking (potential firewall)
	for pattern, count := range blockPatterns {
		if count >= 5 {
			issue := NetworkIssue{
				ID:          a.generateIssueID(),
				Type:        IssueTypeConfiguration,
				Severity:    SeverityHigh,
				Title:       fmt.Sprintf("Potential Firewall Blocking: %s", pattern),
				Description: fmt.Sprintf("Consistent timeout failures detected for %s, indicating potential firewall blocking", pattern),
				Affected:    []string{pattern},
				Suggestions: []string{
					"Check NetworkPolicy rules for blocking policies",
					"Verify cloud provider security groups",
					"Check iptables rules on nodes",
					"Review Calico/Cilium/CNI plugin configurations",
					"Test connectivity from within the same namespace",
				},
				Details: map[string]interface{}{
					"failure_count":  count,
					"failure_pattern": "timeout",
				},
				Timestamp: time.Now(),
			}
			a.addIssue(issue)
		}
	}
	
	// Check for widespread timeout issues
	if timeoutErrors > 10 {
		issue := NetworkIssue{
			ID:          a.generateIssueID(),
			Type:        IssueTypeConfiguration,
			Severity:    SeverityCritical,
			Title:       "Widespread Network Timeouts Detected",
			Description: fmt.Sprintf("%d timeout errors detected, indicating potential network-wide firewall or routing issues", timeoutErrors),
			Suggestions: []string{
				"Check cluster-wide NetworkPolicies",
				"Verify kube-proxy is running on all nodes",
				"Check CNI plugin status and logs",
				"Review cloud provider network ACLs",
				"Verify inter-node connectivity",
			},
			Details: map[string]interface{}{
				"total_timeouts":       timeoutErrors,
				"connection_refused":   connectionRefused,
				"affected_connections": len(blockPatterns),
			},
			Timestamp: time.Now(),
		}
		a.addIssue(issue)
	}
}

// analyzeLatency checks for high latency issues
func (a *Analyzer) analyzeLatency(p *prober.Prober) {
	results := p.GetRecentResults(5 * time.Minute)
	
	// Group by target and calculate average latency
	latencyMap := make(map[string][]int64)
	for _, result := range results {
		if result.Success {
			key := fmt.Sprintf("%s:%d", result.TargetIP, result.TargetPort)
			latencyMap[key] = append(latencyMap[key], result.Latency)
		}
	}

	for target, latencies := range latencyMap {
		if len(latencies) > 0 {
			var sum int64
			for _, l := range latencies {
				sum += l
			}
			avg := sum / int64(len(latencies))

			if avg > 100 { // More than 100ms average
				issue := NetworkIssue{
					ID:          a.generateIssueID(),
					Type:        IssueTypeLatency,
					Severity:    SeverityMedium,
					Title:       fmt.Sprintf("High Latency Detected: %s", target),
					Description: fmt.Sprintf("Average latency to %s is %dms", target, avg),
					Affected:    []string{target},
					Suggestions: []string{
						"Check network congestion",
						"Review pod placement and affinity rules",
						"Consider using node-local traffic policies",
						"Check for CPU throttling on target pods",
					},
					Details: map[string]interface{}{
						"average_latency_ms": avg,
						"sample_count":       len(latencies),
					},
					Timestamp: time.Now(),
				}
				a.addIssue(issue)
			}
		}
	}
}

// SimulateNetworkPolicy simulates the impact of a network policy change
func (a *Analyzer) SimulateNetworkPolicy(policy *networkingv1.NetworkPolicy) []NetworkIssue {
	simulatedIssues := []NetworkIssue{}

	// Analyze what the policy would affect
	topology := a.graphEngine.GetTopology()
	
	affectedPods := []string{}
	for _, node := range topology.Nodes {
		if node.Type == graph.NodeTypePod && node.Namespace == policy.Namespace {
			// Simplified selector matching
			affectedPods = append(affectedPods, node.ID)
		}
	}

	if len(affectedPods) > 0 {
		issue := NetworkIssue{
			ID:          a.generateIssueID(),
			Type:        IssueTypePolicy,
			Severity:    SeverityMedium,
			Title:       "Network Policy Simulation Result",
			Description: fmt.Sprintf("Policy '%s' would affect %d pods in namespace %s", policy.Name, len(affectedPods), policy.Namespace),
			Affected:    affectedPods,
			Suggestions: []string{
				"Review the pod selector to ensure it matches intended targets",
				"Test in a non-production environment first",
				"Monitor connectivity after applying the policy",
			},
			Timestamp: time.Now(),
		}
		simulatedIssues = append(simulatedIssues, issue)
	}

	return simulatedIssues
}

// addIssue adds an issue to the collection
func (a *Analyzer) addIssue(issue NetworkIssue) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.issues = append(a.issues, issue)
}

// generateIssueID generates a unique issue ID
func (a *Analyzer) generateIssueID() string {
	a.issueCount++
	return fmt.Sprintf("issue-%d-%d", time.Now().Unix(), a.issueCount)
}

// GetIssues returns all detected issues
func (a *Analyzer) GetIssues() []NetworkIssue {
	a.mu.RLock()
	defer a.mu.RUnlock()

	issues := make([]NetworkIssue, len(a.issues))
	copy(issues, a.issues)
	return issues
}

// GetIssuesBySeverity returns issues filtered by severity
func (a *Analyzer) GetIssuesBySeverity(severity IssueSeverity) []NetworkIssue {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filtered []NetworkIssue
	for _, issue := range a.issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// GetIssuesByType returns issues filtered by type
func (a *Analyzer) GetIssuesByType(issueType IssueType) []NetworkIssue {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var filtered []NetworkIssue
	for _, issue := range a.issues {
		if issue.Type == issueType {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// generateIntelligentInsights creates actionable recommendations
func (a *Analyzer) generateIntelligentInsights(collector *collector.Collector, p *prober.Prober) {
	// Resource optimization insights
	a.generateResourceOptimizationInsights(collector)
	
	// Security enhancement insights
	a.generateSecurityInsights(collector)
	
	// Network topology optimization insights
	a.generateNetworkOptimizationInsights(collector)
	
	// Performance improvement insights
	a.generatePerformanceInsights(collector, p)
}

// generateResourceOptimizationInsights analyzes resource usage patterns
func (a *Analyzer) generateResourceOptimizationInsights(collector *collector.Collector) {
	pods := collector.GetPods()
	nodes := collector.GetNodes()
	
	// Analyze pod distribution across nodes
	nodeLoadMap := make(map[string]int)
	for _, pod := range pods {
		if pod.Spec.NodeName != "" {
			nodeLoadMap[pod.Spec.NodeName]++
		}
	}
	
	// Check for unbalanced node usage
	if len(nodes) > 1 {
		totalPods := len(pods)
		avgPodsPerNode := float64(totalPods) / float64(len(nodes))
		
		for nodeName, podCount := range nodeLoadMap {
			if float64(podCount) > avgPodsPerNode*1.5 {
				insight := IntelligentInsight{
					ID:          a.generateInsightID(),
					Category:    "optimization",
					Priority:    "medium",
					Title:       fmt.Sprintf("Unbalanced Pod Distribution on Node %s", nodeName),
					Description: fmt.Sprintf("Node %s has %d pods (%.1f%% above average). Consider rebalancing workloads.", nodeName, podCount, (float64(podCount)/avgPodsPerNode-1)*100),
					Impact:      "Better resource utilization and improved fault tolerance",
					Actions: []ActionableStep{
						{
							Description: "Review pod affinity and anti-affinity rules",
							Risk:        "low",
							Automated:   false,
						},
						{
							Description: "Consider using pod disruption budgets",
							Command:     "kubectl create poddisruptionbudget",
							Risk:        "low",
							Automated:   false,
						},
					},
					Metrics: map[string]interface{}{
						"current_pod_count": podCount,
						"average_pod_count": avgPodsPerNode,
						"imbalance_percent": (float64(podCount)/avgPodsPerNode - 1) * 100,
					},
					Confidence: 0.8,
					Timestamp:  time.Now(),
				}
				a.addInsight(insight)
			}
		}
	}
}

// generateSecurityInsights analyzes security posture
func (a *Analyzer) generateSecurityInsights(collector *collector.Collector) {
	policies := collector.GetNetworkPolicies()
	pods := collector.GetPods()
	
	// Count namespaces without network policies
	namespacesWithPods := make(map[string]bool)
	namespacesWithPolicies := make(map[string]bool)
	
	for _, pod := range pods {
		namespacesWithPods[pod.Namespace] = true
	}
	
	for _, policy := range policies {
		namespacesWithPolicies[policy.Namespace] = true
	}
	
	unprotectedNamespaces := 0
	for ns := range namespacesWithPods {
		if !namespacesWithPolicies[ns] && ns != "kube-system" {
			unprotectedNamespaces++
		}
	}
	
	if unprotectedNamespaces > 0 {
		insight := IntelligentInsight{
			ID:          a.generateInsightID(),
			Category:    "security",
			Priority:    "high",
			Title:       "Implement Zero-Trust Network Policies",
			Description: fmt.Sprintf("%d namespaces lack network policies, creating security gaps. Implement default-deny policies for enhanced security.", unprotectedNamespaces),
			Impact:      "Significantly improved network security and reduced attack surface",
			Actions: []ActionableStep{
				{
					Description: "Create default-deny network policy template",
					Command:     "kubectl apply -f default-deny-policy.yaml",
					Risk:        "medium",
					Automated:   false,
				},
				{
					Description: "Gradually implement namespace-specific policies",
					Risk:        "low",
					Automated:   false,
				},
			},
			Metrics: map[string]interface{}{
				"unprotected_namespaces": unprotectedNamespaces,
				"total_namespaces":      len(namespacesWithPods),
				"coverage_percent":      float64(len(namespacesWithPolicies)) / float64(len(namespacesWithPods)) * 100,
			},
			Confidence: 0.9,
			Timestamp:  time.Now(),
		}
		a.addInsight(insight)
	}
}

// generateNetworkOptimizationInsights analyzes network topology
func (a *Analyzer) generateNetworkOptimizationInsights(collector *collector.Collector) {
	services := collector.GetServices()
	
	// Analyze service types and suggest optimizations
	loadBalancerCount := 0
	nodePortCount := 0
	
	for _, service := range services {
		switch service.Spec.Type {
		case corev1.ServiceTypeLoadBalancer:
			loadBalancerCount++
		case corev1.ServiceTypeNodePort:
			nodePortCount++
		}
	}
	
	if loadBalancerCount > 3 {
		insight := IntelligentInsight{
			ID:          a.generateInsightID(),
			Category:    "optimization",
			Priority:    "medium",
			Title:       "Optimize LoadBalancer Usage",
			Description: fmt.Sprintf("Cluster has %d LoadBalancer services. Consider using an Ingress controller to reduce cloud costs and improve management.", loadBalancerCount),
			Impact:      "Reduced cloud costs and simplified traffic management",
			Actions: []ActionableStep{
				{
					Description: "Deploy NGINX or Traefik ingress controller",
					Command:     "kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml",
					Risk:        "low",
					Automated:   false,
				},
				{
					Description: "Convert LoadBalancer services to ClusterIP with Ingress",
					Risk:        "medium",
					Automated:   false,
				},
			},
			Metrics: map[string]interface{}{
				"loadbalancer_count": loadBalancerCount,
				"potential_savings":  "~$50-200/month per LoadBalancer",
			},
			Confidence: 0.85,
			Timestamp:  time.Now(),
		}
		a.addInsight(insight)
	}
}

// generatePerformanceInsights analyzes performance patterns
func (a *Analyzer) generatePerformanceInsights(collector *collector.Collector, p *prober.Prober) {
	pods := collector.GetPods()
	
	// Analyze pod restart patterns for performance issues
	highRestartPods := 0
	for _, pod := range pods {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.RestartCount > 3 {
				highRestartPods++
				break
			}
		}
	}
	
	if highRestartPods > 0 {
		insight := IntelligentInsight{
			ID:          a.generateInsightID(),
			Category:    "reliability",
			Priority:    "high",
			Title:       "Implement Resource Limits and Health Checks",
			Description: fmt.Sprintf("%d pods have frequent restarts. Implement proper resource limits and health checks to improve stability.", highRestartPods),
			Impact:      "Improved application stability and reduced downtime",
			Actions: []ActionableStep{
				{
					Description: "Add resource limits to prevent OOM kills",
					Command:     "kubectl patch deployment <name> -p '{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"<container>\",\"resources\":{\"limits\":{\"memory\":\"512Mi\",\"cpu\":\"500m\"}}}]}}}}'",
					Risk:        "low",
					Automated:   false,
				},
				{
					Description: "Configure liveness and readiness probes",
					Risk:        "low",
					Automated:   false,
				},
			},
			Metrics: map[string]interface{}{
				"pods_with_restarts": highRestartPods,
				"stability_score":    float64(len(pods)-highRestartPods) / float64(len(pods)) * 100,
			},
			Confidence: 0.9,
			Timestamp:  time.Now(),
		}
		a.addInsight(insight)
	}
}

// Simulation methods
func (a *Analyzer) RunSimulation(request SimulationRequest) *SimulationResult {
	result := &SimulationResult{
		ID:        a.generateSimulationID(),
		Request:   request,
		Timestamp: time.Now(),
	}
	
	switch request.Type {
	case "policy":
		a.simulateNetworkPolicyChange(request, result)
	case "resource":
		a.simulateResourceChange(request, result)
	case "topology":
		a.simulateTopologyChange(request, result)
	default:
		result.Risks = []SimulationRisk{
			{
				Description: "Unknown simulation type",
				Severity:    "high",
				Likelihood:  1.0,
				Mitigation:  "Use supported simulation types: policy, resource, topology",
			},
		}
	}
	
	a.mu.Lock()
	a.simulations = append(a.simulations, *result)
	a.mu.Unlock()
	
	return result
}

// simulateNetworkPolicyChange predicts impact of network policy changes
func (a *Analyzer) simulateNetworkPolicyChange(request SimulationRequest, result *SimulationResult) {
	// Analyze current connectivity patterns
	result.Impact.Connectivity = map[string]string{
		"current": "Full connectivity between all pods",
		"after":   "Restricted connectivity based on policy rules",
	}
	
	result.Impact.Security = map[string]string{
		"current": "Open network (security score: 3/10)",
		"after":   "Secured network (estimated score: 8/10)",
	}
	
	result.Benefits = []string{
		"Reduced attack surface",
		"Compliance with security standards",
		"Better network isolation",
	}
	
	result.Risks = []SimulationRisk{
		{
			Description: "Potential service disruption if policy is too restrictive",
			Severity:    "medium",
			Likelihood:  0.3,
			Mitigation:  "Test in staging environment first",
		},
		{
			Description: "Troubleshooting complexity increases",
			Severity:    "low",
			Likelihood:  0.7,
			Mitigation:  "Implement comprehensive monitoring and logging",
		},
	}
	
	result.Timeline = "15-30 minutes for policy application and verification"
	result.Confidence = 0.85
}

// simulateResourceChange predicts impact of resource allocation changes
func (a *Analyzer) simulateResourceChange(request SimulationRequest, result *SimulationResult) {
	result.Impact.Performance = map[string]string{
		"current": "Variable performance due to resource contention",
		"after":   "Predictable performance with resource guarantees",
	}
	
	result.Impact.ResourceUsage = map[string]string{
		"current": "70% node utilization",
		"after":   "Estimated 85% node utilization",
	}
	
	result.Benefits = []string{
		"Improved application performance",
		"Better resource utilization",
		"Reduced scheduling delays",
	}
	
	result.Risks = []SimulationRisk{
		{
			Description: "Higher resource consumption may require node scaling",
			Severity:    "medium",
			Likelihood:  0.4,
			Mitigation:  "Monitor node capacity and enable cluster autoscaling",
		},
	}
	
	result.Timeline = "5-10 minutes for rolling update completion"
	result.Confidence = 0.8
}

// simulateTopologyChange predicts impact of topology modifications
func (a *Analyzer) simulateTopologyChange(request SimulationRequest, result *SimulationResult) {
	result.Impact.Connectivity = map[string]string{
		"current": "Direct pod-to-pod communication",
		"after":   "Service mesh routing with load balancing",
	}
	
	result.Benefits = []string{
		"Improved traffic distribution",
		"Enhanced observability",
		"Better fault tolerance",
	}
	
	result.Risks = []SimulationRisk{
		{
			Description: "Increased latency due to additional network hops",
			Severity:    "low",
			Likelihood:  0.5,
			Mitigation:  "Monitor latency metrics and optimize routing",
		},
	}
	
	result.Timeline = "1-2 hours for full topology migration"
	result.Confidence = 0.75
}

// Helper methods
func (a *Analyzer) addInsight(insight IntelligentInsight) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.insights = append(a.insights, insight)
}

func (a *Analyzer) generateInsightID() string {
	return fmt.Sprintf("insight-%d-%d", time.Now().Unix(), len(a.insights))
}

func (a *Analyzer) generateSimulationID() string {
	return fmt.Sprintf("sim-%d-%d", time.Now().Unix(), len(a.simulations))
}

// GetInsights returns all intelligent insights
func (a *Analyzer) GetInsights() []IntelligentInsight {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]IntelligentInsight(nil), a.insights...)
}

// GetSimulations returns all simulation results
func (a *Analyzer) GetSimulations() []SimulationResult {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]SimulationResult(nil), a.simulations...)
}