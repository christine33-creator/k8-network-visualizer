package simulator

import (
	"fmt"
	"strings"

	"github.com/christine33-creator/k8-network-visualizer/pkg/ai"
	"github.com/christine33-creator/k8-network-visualizer/pkg/graph"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SimulationType represents the type of simulation
type SimulationType string

const (
	SimulationTypeNetworkPolicy SimulationType = "network_policy"
	SimulationTypePodFailure    SimulationType = "pod_failure"
	SimulationTypeNodeFailure   SimulationType = "node_failure"
	SimulationTypeServiceMesh   SimulationType = "service_mesh"
)

// SimulationResult represents the outcome of a simulation
type SimulationResult struct {
	Type            SimulationType         `json:"type"`
	Impact          ImpactAnalysis         `json:"impact"`
	AffectedFlows   []NetworkFlow          `json:"affected_flows"`
	Recommendations []string               `json:"recommendations"`
	RiskLevel       string                 `json:"risk_level"`
	Summary         string                 `json:"summary"`
	AIAnalysis      string                 `json:"ai_analysis,omitempty"`
}

// ImpactAnalysis describes the impact of a change
type ImpactAnalysis struct {
	TotalPodsAffected     int      `json:"total_pods_affected"`
	TotalServicesAffected int      `json:"total_services_affected"`
	BlockedConnections    int      `json:"blocked_connections"`
	NewConnections        int      `json:"new_connections"`
	AffectedNamespaces    []string `json:"affected_namespaces"`
	CriticalPathsImpacted []string `json:"critical_paths_impacted"`
}

// NetworkFlow represents a network connection flow
type NetworkFlow struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Protocol    string `json:"protocol"`
	Port        int32  `json:"port"`
	CurrentState string `json:"current_state"`
	NewState    string `json:"new_state"`
	Impact      string `json:"impact"`
}

// Simulator performs what-if analysis
type Simulator struct {
	graphEngine *graph.Engine
	pods        map[string]*corev1.Pod
	services    map[string]*corev1.Service
	policies    map[string]*networkingv1.NetworkPolicy
	aiClient    *ai.Client
}

// NewSimulator creates a new simulator instance
func NewSimulator(engine *graph.Engine) *Simulator {
	return &Simulator{
		graphEngine: engine,
		pods:        make(map[string]*corev1.Pod),
		services:    make(map[string]*corev1.Service),
		policies:    make(map[string]*networkingv1.NetworkPolicy),
		aiClient:    nil, // Will be set when API key is provided
	}
}

// SetAIClient sets the AI client for enhanced analysis
func (s *Simulator) SetAIClient(client *ai.Client) {
	s.aiClient = client
}

// UpdateResources updates the simulator's view of cluster resources
func (s *Simulator) UpdateResources(pods []*corev1.Pod, services []*corev1.Service, policies []*networkingv1.NetworkPolicy) {
	// Update pods
	s.pods = make(map[string]*corev1.Pod)
	for _, pod := range pods {
		key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		s.pods[key] = pod
	}

	// Update services
	s.services = make(map[string]*corev1.Service)
	for _, svc := range services {
		key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
		s.services[key] = svc
	}

	// Update policies
	s.policies = make(map[string]*networkingv1.NetworkPolicy)
	for _, policy := range policies {
		key := fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)
		s.policies[key] = policy
	}
}

// SimulateNetworkPolicy simulates the impact of adding/modifying a NetworkPolicy
func (s *Simulator) SimulateNetworkPolicy(policy *networkingv1.NetworkPolicy, action string) (*SimulationResult, error) {
	result := &SimulationResult{
		Type: SimulationTypeNetworkPolicy,
		Impact: ImpactAnalysis{
			AffectedNamespaces: []string{},
		},
		AffectedFlows:   []NetworkFlow{},
		Recommendations: []string{},
	}

	// Find pods affected by this policy
	affectedPods := s.findPodsMatchingSelector(policy.Namespace, &policy.Spec.PodSelector)
	result.Impact.TotalPodsAffected = len(affectedPods)

	// Analyze current connectivity
	currentFlows := s.analyzeCurrentFlows(affectedPods)

	// Simulate new connectivity with policy
	newFlows := s.simulateFlowsWithPolicy(affectedPods, policy)

	// Compare flows and identify changes
	for key, currentFlow := range currentFlows {
		newFlow, exists := newFlows[key]
		if !exists || newFlow.State != currentFlow.State {
			flow := NetworkFlow{
				Source:       currentFlow.Source,
				Destination:  currentFlow.Destination,
				Protocol:     currentFlow.Protocol,
				Port:         currentFlow.Port,
				CurrentState: currentFlow.State,
				NewState:     "blocked",
				Impact:       "Connection will be blocked",
			}
			if exists {
				flow.NewState = newFlow.State
				if newFlow.State == "allowed" {
					flow.Impact = "Connection remains allowed"
				}
			}
			result.AffectedFlows = append(result.AffectedFlows, flow)
			if flow.NewState == "blocked" {
				result.Impact.BlockedConnections++
			}
		}
	}

	// Analyze impact on services
	result.Impact.TotalServicesAffected = s.countAffectedServices(affectedPods)

	// Identify critical paths
	result.Impact.CriticalPathsImpacted = s.identifyCriticalPaths(result.AffectedFlows)

	// Generate risk assessment
	result.RiskLevel = s.assessRisk(result.Impact)

	// Generate recommendations
	result.Recommendations = s.generatePolicyRecommendations(policy, result.Impact)

	// Create summary
	result.Summary = fmt.Sprintf(
		"NetworkPolicy '%s' in namespace '%s' will affect %d pods and potentially block %d connections. Risk level: %s",
		policy.Name, policy.Namespace, result.Impact.TotalPodsAffected, result.Impact.BlockedConnections, result.RiskLevel,
	)

	// Generate AI analysis if available
	if s.aiClient != nil {
		context := fmt.Sprintf(`Current State:
- Namespace: %s
- Affected Pods: %d
- Blocked Connections: %d
- Affected Services: %d
- Critical Paths Impacted: %d

Policy Details:
- Name: %s
- Policy Types: %v
- Ingress Rules: %d
- Egress Rules: %d`,
			policy.Namespace,
			result.Impact.TotalPodsAffected,
			result.Impact.BlockedConnections,
			result.Impact.TotalServicesAffected,
			len(result.Impact.CriticalPathsImpacted),
			policy.Name,
			policy.Spec.PolicyTypes,
			len(policy.Spec.Ingress),
			len(policy.Spec.Egress))

		analysis, err := s.aiClient.GenerateScenarioAnalysis(
			"NetworkPolicy Change",
			fmt.Sprintf("Add/modify NetworkPolicy '%s' in namespace '%s'", policy.Name, policy.Namespace),
			context,
		)
		if err == nil {
			result.AIAnalysis = analysis
		}
	}

	return result, nil
}

// SimulatePodFailure simulates the impact of pod failures
func (s *Simulator) SimulatePodFailure(podName, namespace string) (*SimulationResult, error) {
	result := &SimulationResult{
		Type: SimulationTypePodFailure,
		Impact: ImpactAnalysis{
			AffectedNamespaces: []string{namespace},
		},
		AffectedFlows:   []NetworkFlow{},
		Recommendations: []string{},
	}

	podKey := fmt.Sprintf("%s/%s", namespace, podName)
	pod, exists := s.pods[podKey]
	if !exists {
		return nil, fmt.Errorf("pod %s not found", podKey)
	}

	// Find services that depend on this pod
	affectedServices := s.findServicesForPod(pod)
	result.Impact.TotalServicesAffected = len(affectedServices)

	// Check replica count for resilience
	replicas := s.countReplicas(pod)
	if replicas <= 1 {
		result.RiskLevel = "critical"
		result.Recommendations = append(result.Recommendations, 
			"CRITICAL: This is the only replica! Service will be unavailable.",
			"Increase replica count to at least 3 for high availability.",
		)
	} else if replicas == 2 {
		result.RiskLevel = "high"
		result.Recommendations = append(result.Recommendations,
			"Only one replica will remain. Service degradation possible.",
			"Consider increasing replica count for better resilience.",
		)
	} else {
		result.RiskLevel = "low"
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("%d replicas will remain. Service should continue normally.", replicas-1),
		)
	}

	// Identify affected network flows
	flows := s.findFlowsInvolvingPod(pod)
	for _, flow := range flows {
		flow.NewState = "rerouted"
		flow.Impact = fmt.Sprintf("Traffic will be rerouted to remaining %d replicas", replicas-1)
		if replicas <= 1 {
			flow.NewState = "failed"
			flow.Impact = "Connection will fail - no remaining replicas"
		}
		result.AffectedFlows = append(result.AffectedFlows, flow)
	}

	result.Summary = fmt.Sprintf(
		"Failure of pod '%s' will affect %d services. %d replicas will remain. Risk level: %s",
		podName, len(affectedServices), replicas-1, result.RiskLevel,
	)

	return result, nil
}

// SimulateNodeFailure simulates the impact of node failures
func (s *Simulator) SimulateNodeFailure(nodeName string) (*SimulationResult, error) {
	result := &SimulationResult{
		Type: SimulationTypeNodeFailure,
		Impact: ImpactAnalysis{
			AffectedNamespaces: []string{},
		},
		AffectedFlows:   []NetworkFlow{},
		Recommendations: []string{},
	}

	// Find all pods on this node
	podsOnNode := s.findPodsOnNode(nodeName)
	result.Impact.TotalPodsAffected = len(podsOnNode)

	// Group pods by deployment/service
	deploymentMap := make(map[string][]*corev1.Pod)
	for _, pod := range podsOnNode {
		owner := s.getOwnerReference(pod)
		deploymentMap[owner] = append(deploymentMap[owner], pod)
	}

	// Analyze impact per deployment
	criticalServices := []string{}
	for deployment, pods := range deploymentMap {
		totalReplicas := s.countReplicasForDeployment(deployment)
		affectedReplicas := len(pods)
		remainingReplicas := totalReplicas - affectedReplicas

		if remainingReplicas == 0 {
			criticalServices = append(criticalServices, deployment)
			result.Impact.CriticalPathsImpacted = append(result.Impact.CriticalPathsImpacted, deployment)
		}

		// Create flow impact
		for _, pod := range pods {
			flows := s.findFlowsInvolvingPod(pod)
			for _, flow := range flows {
				if remainingReplicas > 0 {
					flow.NewState = "degraded"
					flow.Impact = fmt.Sprintf("Reduced capacity: %d/%d replicas remaining", remainingReplicas, totalReplicas)
				} else {
					flow.NewState = "failed"
					flow.Impact = "Service unavailable - all replicas on failed node"
				}
				result.AffectedFlows = append(result.AffectedFlows, flow)
			}
		}
	}

	// Assess overall risk
	if len(criticalServices) > 0 {
		result.RiskLevel = "critical"
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("CRITICAL: %d services will be completely unavailable!", len(criticalServices)),
			"Enable pod anti-affinity rules to spread replicas across nodes.",
			"Increase replica counts for critical services.",
		)
	} else if result.Impact.TotalPodsAffected > 10 {
		result.RiskLevel = "high"
		result.Recommendations = append(result.Recommendations,
			"Multiple services will operate at reduced capacity.",
			"Consider using pod disruption budgets.",
		)
	} else {
		result.RiskLevel = "medium"
		result.Recommendations = append(result.Recommendations,
			"Services have sufficient replicas on other nodes.",
			"Monitor for increased load on remaining nodes.",
		)
	}

	result.Summary = fmt.Sprintf(
		"Node '%s' failure will affect %d pods across %d deployments. %d services will be critical. Risk level: %s",
		nodeName, result.Impact.TotalPodsAffected, len(deploymentMap), len(criticalServices), result.RiskLevel,
	)

	return result, nil
}

// SimulateServiceMesh simulates service mesh policy changes (Istio/Linkerd)
func (s *Simulator) SimulateServiceMesh(policy ServiceMeshPolicy) (*SimulationResult, error) {
	result := &SimulationResult{
		Type: SimulationTypeServiceMesh,
		Impact: ImpactAnalysis{
			AffectedNamespaces: []string{},
		},
		AffectedFlows:   []NetworkFlow{},
		Recommendations: []string{},
	}

	// Simulate different service mesh scenarios
	switch policy.Type {
	case "traffic-split":
		result = s.simulateTrafficSplit(policy)
	case "circuit-breaker":
		result = s.simulateCircuitBreaker(policy)
	case "retry-policy":
		result = s.simulateRetryPolicy(policy)
	case "timeout":
		result = s.simulateTimeout(policy)
	}

	return result, nil
}

// Helper methods

func (s *Simulator) findPodsMatchingSelector(namespace string, selector *metav1.LabelSelector) []*corev1.Pod {
	var matched []*corev1.Pod
	
	for _, pod := range s.pods {
		if pod.Namespace != namespace {
			continue
		}
		
		if selector == nil || len(selector.MatchLabels) == 0 {
			matched = append(matched, pod)
			continue
		}
		
		if s.labelsMatch(pod.Labels, selector.MatchLabels) {
			matched = append(matched, pod)
		}
	}
	
	return matched
}

func (s *Simulator) labelsMatch(podLabels, selectorLabels map[string]string) bool {
	for key, value := range selectorLabels {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}

func (s *Simulator) analyzeCurrentFlows(pods []*corev1.Pod) map[string]*Flow {
	flows := make(map[string]*Flow)
	
	for _, pod := range pods {
		// Find services this pod connects to
		for _, svc := range s.services {
			flowKey := fmt.Sprintf("%s/%s->%s/%s", pod.Namespace, pod.Name, svc.Namespace, svc.Name)
			flows[flowKey] = &Flow{
				Source:      fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Destination: fmt.Sprintf("%s/%s", svc.Namespace, svc.Name),
				Protocol:    "TCP",
				State:       "allowed",
			}
		}
	}
	
	return flows
}

func (s *Simulator) simulateFlowsWithPolicy(pods []*corev1.Pod, policy *networkingv1.NetworkPolicy) map[string]*Flow {
	flows := make(map[string]*Flow)
	
	for _, pod := range pods {
		for _, svc := range s.services {
			flowKey := fmt.Sprintf("%s/%s->%s/%s", pod.Namespace, pod.Name, svc.Namespace, svc.Name)
			
			// Check if flow is allowed by policy
			allowed := s.isPolicyAllowed(pod, svc, policy)
			state := "blocked"
			if allowed {
				state = "allowed"
			}
			
			flows[flowKey] = &Flow{
				Source:      fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Destination: fmt.Sprintf("%s/%s", svc.Namespace, svc.Name),
				Protocol:    "TCP",
				State:       state,
			}
		}
	}
	
	return flows
}

func (s *Simulator) isPolicyAllowed(pod *corev1.Pod, svc *corev1.Service, policy *networkingv1.NetworkPolicy) bool {
	// Simplified policy evaluation
	if policy.Spec.PolicyTypes == nil || len(policy.Spec.PolicyTypes) == 0 {
		return true
	}
	
	for _, policyType := range policy.Spec.PolicyTypes {
		if policyType == networkingv1.PolicyTypeEgress {
			// Check egress rules
			if policy.Spec.Egress == nil || len(policy.Spec.Egress) == 0 {
				return false // Default deny
			}
			
			// Check if any egress rule matches
			for _, rule := range policy.Spec.Egress {
				if s.egressRuleMatches(rule, svc) {
					return true
				}
			}
			return false
		}
	}
	
	return true
}

func (s *Simulator) egressRuleMatches(rule networkingv1.NetworkPolicyEgressRule, svc *corev1.Service) bool {
	// Simplified rule matching
	if rule.To == nil || len(rule.To) == 0 {
		return true // Allow all
	}
	
	for _, to := range rule.To {
		if to.PodSelector != nil {
			// Check if service endpoints match selector
			return true // Simplified
		}
	}
	
	return false
}

func (s *Simulator) countAffectedServices(pods []*corev1.Pod) int {
	affectedServices := make(map[string]bool)
	
	for _, pod := range pods {
		for _, svc := range s.services {
			if svc.Namespace == pod.Namespace {
				if s.labelsMatch(pod.Labels, svc.Spec.Selector) {
					key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
					affectedServices[key] = true
				}
			}
		}
	}
	
	return len(affectedServices)
}

func (s *Simulator) identifyCriticalPaths(flows []NetworkFlow) []string {
	critical := []string{}
	
	for _, flow := range flows {
		if flow.NewState == "blocked" {
			// Check if this is a critical service
			if strings.Contains(flow.Destination, "database") || 
			   strings.Contains(flow.Destination, "auth") ||
			   strings.Contains(flow.Destination, "payment") {
				critical = append(critical, fmt.Sprintf("%s -> %s", flow.Source, flow.Destination))
			}
		}
	}
	
	return critical
}

func (s *Simulator) assessRisk(impact ImpactAnalysis) string {
	if len(impact.CriticalPathsImpacted) > 0 {
		return "critical"
	}
	if impact.BlockedConnections > 10 || impact.TotalPodsAffected > 20 {
		return "high"
	}
	if impact.BlockedConnections > 5 || impact.TotalPodsAffected > 10 {
		return "medium"
	}
	return "low"
}

func (s *Simulator) generatePolicyRecommendations(policy *networkingv1.NetworkPolicy, impact ImpactAnalysis) []string {
	recommendations := []string{}
	
	if impact.BlockedConnections > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("This policy will block %d existing connections.", impact.BlockedConnections),
			"Review the blocked connections to ensure they are intentional.",
		)
	}
	
	if len(impact.CriticalPathsImpacted) > 0 {
		recommendations = append(recommendations,
			"CRITICAL: This policy affects critical service paths!",
			"Consider adding explicit allow rules for critical services.",
		)
	}
	
	if policy.Spec.Egress == nil || len(policy.Spec.Egress) == 0 {
		recommendations = append(recommendations,
			"No egress rules defined - all outbound traffic will be blocked.",
			"Add egress rules for DNS and required services.",
		)
	}
	
	return recommendations
}

func (s *Simulator) findServicesForPod(pod *corev1.Pod) []*corev1.Service {
	var services []*corev1.Service
	
	for _, svc := range s.services {
		if svc.Namespace == pod.Namespace {
			if s.labelsMatch(pod.Labels, svc.Spec.Selector) {
				services = append(services, svc)
			}
		}
	}
	
	return services
}

func (s *Simulator) countReplicas(pod *corev1.Pod) int {
	count := 0
	
	for _, p := range s.pods {
		if p.Namespace == pod.Namespace {
			// Check if same deployment
			if s.getOwnerReference(p) == s.getOwnerReference(pod) {
				count++
			}
		}
	}
	
	return count
}

func (s *Simulator) findFlowsInvolvingPod(pod *corev1.Pod) []NetworkFlow {
	var flows []NetworkFlow
	
	podID := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	topology := s.graphEngine.GetTopology()
	
	for _, edge := range topology.Edges {
		if edge.Source == podID || edge.Target == podID {
			flows = append(flows, NetworkFlow{
				Source:       edge.Source,
				Destination:  edge.Target,
				Protocol:     "TCP",
				CurrentState: string(edge.Health),
			})
		}
	}
	
	return flows
}

func (s *Simulator) findPodsOnNode(nodeName string) []*corev1.Pod {
	var pods []*corev1.Pod
	
	for _, pod := range s.pods {
		if pod.Spec.NodeName == nodeName {
			pods = append(pods, pod)
		}
	}
	
	return pods
}

func (s *Simulator) getOwnerReference(pod *corev1.Pod) string {
	if len(pod.OwnerReferences) > 0 {
		return fmt.Sprintf("%s/%s", pod.Namespace, pod.OwnerReferences[0].Name)
	}
	return fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
}

func (s *Simulator) countReplicasForDeployment(deployment string) int {
	count := 0
	
	for _, pod := range s.pods {
		if s.getOwnerReference(pod) == deployment {
			count++
		}
	}
	
	return count
}

func (s *Simulator) simulateTrafficSplit(policy ServiceMeshPolicy) *SimulationResult {
	result := &SimulationResult{
		Type:     SimulationTypeServiceMesh,
		RiskLevel: "low",
		Summary:  fmt.Sprintf("Traffic split: %d%% to v1, %d%% to v2", policy.WeightV1, policy.WeightV2),
	}
	
	result.Recommendations = append(result.Recommendations,
		"Monitor error rates during traffic shift.",
		"Ensure both versions have sufficient capacity.",
		"Set up proper observability for both versions.",
	)
	
	return result
}

func (s *Simulator) simulateCircuitBreaker(policy ServiceMeshPolicy) *SimulationResult {
	result := &SimulationResult{
		Type:     SimulationTypeServiceMesh,
		RiskLevel: "medium",
		Summary:  fmt.Sprintf("Circuit breaker: Open after %d consecutive failures", policy.ConsecutiveErrors),
	}
	
	result.Recommendations = append(result.Recommendations,
		"Ensure fallback behavior is properly configured.",
		"Monitor circuit breaker metrics.",
		"Test circuit breaker behavior under load.",
	)
	
	return result
}

func (s *Simulator) simulateRetryPolicy(policy ServiceMeshPolicy) *SimulationResult {
	result := &SimulationResult{
		Type:     SimulationTypeServiceMesh,
		RiskLevel: "low",
		Summary:  fmt.Sprintf("Retry policy: %d attempts with %s backoff", policy.Attempts, policy.BackoffInterval),
	}
	
	if policy.Attempts > 5 {
		result.RiskLevel = "medium"
		result.Recommendations = append(result.Recommendations,
			"High retry count may cause cascading failures.",
			"Consider implementing exponential backoff.",
		)
	}
	
	return result
}

func (s *Simulator) simulateTimeout(policy ServiceMeshPolicy) *SimulationResult {
	result := &SimulationResult{
		Type:     SimulationTypeServiceMesh,
		RiskLevel: "medium",
		Summary:  fmt.Sprintf("Request timeout set to %s", policy.Timeout),
	}
	
	result.Recommendations = append(result.Recommendations,
		"Ensure timeout is appropriate for service SLA.",
		"Consider different timeouts for different operations.",
		"Monitor timeout metrics and adjust as needed.",
	)
	
	return result
}

// Flow represents a network flow
type Flow struct {
	Source      string
	Destination string
	Protocol    string
	Port        int32
	State       string
}

// ServiceMeshPolicy represents a service mesh policy
type ServiceMeshPolicy struct {
	Type              string `json:"type"`
	Service           string `json:"service"`
	WeightV1          int    `json:"weight_v1,omitempty"`
	WeightV2          int    `json:"weight_v2,omitempty"`
	ConsecutiveErrors int    `json:"consecutive_errors,omitempty"`
	Attempts          int    `json:"attempts,omitempty"`
	BackoffInterval   string `json:"backoff_interval,omitempty"`
	Timeout           string `json:"timeout,omitempty"`
}