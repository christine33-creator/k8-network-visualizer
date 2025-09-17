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

// Analyzer performs network analysis and issue detection
type Analyzer struct {
	graphEngine *graph.Engine
	issues      []NetworkIssue
	mu          sync.RWMutex
	issueCount  int
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(engine *graph.Engine) *Analyzer {
	return &Analyzer{
		graphEngine: engine,
		issues:      make([]NetworkIssue, 0),
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
func (a *Analyzer) analyze(collector *collector.Collector, prober *prober.Prober) {
	// Clear previous issues
	a.mu.Lock()
	a.issues = make([]NetworkIssue, 0)
	a.mu.Unlock()

	// Update graph with latest data
	a.updateGraph(collector)

	// Perform various analyses
	a.analyzeConnectivity(prober)
	a.analyzeNetworkPolicies(collector)
	a.analyzePodHealth(collector)
	a.analyzeServiceEndpoints(collector)
	a.analyzeCIDROverlaps(collector)
	a.analyzeLatency(prober)
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
func (a *Analyzer) analyzeConnectivity(prober *prober.Prober) {
	failedProbes := prober.GetFailedProbes()
	
	// Group failed probes by target
	failureMap := make(map[string][]prober.ProbeResult)
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

// analyzeLatency checks for high latency issues
func (a *Analyzer) analyzeLatency(prober *prober.Prober) {
	results := prober.GetRecentResults(5 * time.Minute)
	
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