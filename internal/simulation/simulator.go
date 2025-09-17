package simulation

import (
	"fmt"
	"strings"

	"github.com/christine33-creator/k8-network-visualizer/pkg/models"
)

// Simulator handles "what if" network scenarios
type Simulator struct{}

// NewSimulator creates a new simulator
func NewSimulator() *Simulator {
	return &Simulator{}
}

// SimulateNetworkPolicyAddition simulates adding a new network policy
func (s *Simulator) SimulateNetworkPolicyAddition(topology *models.NetworkTopology, policy *models.NetworkPolicy) *models.Simulation {
	simulation := &models.Simulation{
		Name:        fmt.Sprintf("Add Network Policy: %s/%s", policy.Namespace, policy.Name),
		Description: "Simulate the effect of adding a new network policy",
		Changes: []models.SimulationChange{
			{
				Type:   "add_policy",
				Target: fmt.Sprintf("%s/%s", policy.Namespace, policy.Name),
				Data:   policy,
			},
		},
	}

	// Analyze impact on existing connections
	for _, connection := range topology.Connections {
		result := s.analyzeConnectionWithPolicy(connection, policy, topology)
		simulation.Results = append(simulation.Results, result)
	}

	return simulation
}

// SimulateNetworkPolicyRemoval simulates removing an existing network policy
func (s *Simulator) SimulateNetworkPolicyRemoval(topology *models.NetworkTopology, policyName, namespace string) *models.Simulation {
	simulation := &models.Simulation{
		Name:        fmt.Sprintf("Remove Network Policy: %s/%s", namespace, policyName),
		Description: "Simulate the effect of removing an existing network policy",
		Changes: []models.SimulationChange{
			{
				Type:   "remove_policy",
				Target: fmt.Sprintf("%s/%s", namespace, policyName),
			},
		},
	}

	// Find the policy to remove
	var targetPolicy *models.NetworkPolicy
	for _, policy := range topology.Policies {
		if policy.Name == policyName && policy.Namespace == namespace {
			targetPolicy = &policy
			break
		}
	}

	if targetPolicy == nil {
		return simulation
	}

	// Analyze impact on existing connections
	for _, connection := range topology.Connections {
		result := s.analyzeConnectionWithoutPolicy(connection, *targetPolicy, topology)
		simulation.Results = append(simulation.Results, result)
	}

	return simulation
}

// SimulatePortBlocking simulates blocking a specific port
func (s *Simulator) SimulatePortBlocking(topology *models.NetworkTopology, port int32, protocol string) *models.Simulation {
	simulation := &models.Simulation{
		Name:        fmt.Sprintf("Block Port %d/%s", port, protocol),
		Description: fmt.Sprintf("Simulate blocking all traffic on port %d/%s", port, protocol),
		Changes: []models.SimulationChange{
			{
				Type:   "block_port",
				Target: fmt.Sprintf("%d/%s", port, protocol),
				Data:   map[string]interface{}{"port": port, "protocol": protocol},
			},
		},
	}

	// Analyze impact on connections using this port
	for _, connection := range topology.Connections {
		var result models.SimulationResult
		result.Connection = connection
		result.Before = connection.Status

		if connection.Port == port && strings.EqualFold(connection.Protocol, protocol) {
			result.After = "blocked"
			result.Impact = "blocked"
		} else {
			result.After = connection.Status
			result.Impact = "no_change"
		}

		simulation.Results = append(simulation.Results, result)
	}

	return simulation
}

// SimulatePodFailure simulates a pod failure
func (s *Simulator) SimulatePodFailure(topology *models.NetworkTopology, podName, namespace string) *models.Simulation {
	simulation := &models.Simulation{
		Name:        fmt.Sprintf("Pod Failure: %s/%s", namespace, podName),
		Description: "Simulate the effect of a pod failure on network connectivity",
		Changes: []models.SimulationChange{
			{
				Type:   "pod_failure",
				Target: fmt.Sprintf("%s/%s", namespace, podName),
			},
		},
	}

	// Find the failing pod
	var failingPod *models.Pod
	for _, pod := range topology.Pods {
		if pod.Name == podName && pod.Namespace == namespace {
			failingPod = &pod
			break
		}
	}

	if failingPod == nil {
		return simulation
	}

	// Analyze impact on connections involving this pod
	for _, connection := range topology.Connections {
		var result models.SimulationResult
		result.Connection = connection
		result.Before = connection.Status

		podIP := failingPod.IP
		if connection.Source == podIP || connection.Destination == podIP {
			result.After = "failed"
			result.Impact = "blocked"
		} else {
			result.After = connection.Status
			result.Impact = "no_change"
		}

		simulation.Results = append(simulation.Results, result)
	}

	return simulation
}

func (s *Simulator) analyzeConnectionWithPolicy(connection models.Connection, policy *models.NetworkPolicy, topology *models.NetworkTopology) models.SimulationResult {
	result := models.SimulationResult{
		Connection: connection,
		Before:     connection.Status,
	}

	// Find source and destination pods
	var sourcePod, destPod *models.Pod
	for _, pod := range topology.Pods {
		if pod.IP == connection.Source {
			sourcePod = &pod
		}
		if pod.IP == connection.Destination {
			destPod = &pod
		}
	}

	// Simple policy evaluation (ingress rules)
	if destPod != nil && s.podMatchesSelector(destPod, policy.Selector) {
		allowed := false

		// Check if connection is allowed by ingress rules
		for _, rule := range policy.Ingress {
			if s.connectionMatchesRule(connection, rule, sourcePod, destPod) {
				allowed = true
				break
			}
		}

		if !allowed {
			result.After = "blocked"
			result.Impact = "blocked"
		} else {
			result.After = "allowed"
			result.Impact = "allowed"
		}
	} else {
		result.After = connection.Status
		result.Impact = "no_change"
	}

	return result
}

func (s *Simulator) analyzeConnectionWithoutPolicy(connection models.Connection, policy models.NetworkPolicy, topology *models.NetworkTopology) models.SimulationResult {
	result := models.SimulationResult{
		Connection: connection,
		Before:     connection.Status,
	}

	// Find destination pod
	var destPod *models.Pod
	for _, pod := range topology.Pods {
		if pod.IP == connection.Destination {
			destPod = &pod
			break
		}
	}

	// If the connection was previously blocked by this policy, it would become allowed
	if destPod != nil && s.podMatchesSelector(destPod, policy.Selector) {
		// Simplified: assume removing policy allows previously blocked connections
		if connection.Status == "blocked" {
			result.After = "allowed"
			result.Impact = "allowed"
		} else {
			result.After = connection.Status
			result.Impact = "no_change"
		}
	} else {
		result.After = connection.Status
		result.Impact = "no_change"
	}

	return result
}

func (s *Simulator) podMatchesSelector(pod *models.Pod, selector map[string]string) bool {
	for key, value := range selector {
		if podValue, exists := pod.Labels[key]; !exists || podValue != value {
			return false
		}
	}
	return true
}

func (s *Simulator) connectionMatchesRule(connection models.Connection, rule models.Rule, sourcePod, destPod *models.Pod) bool {
	// Check if port matches
	portMatches := false
	if len(rule.Ports) == 0 {
		portMatches = true // No port restriction
	} else {
		for _, port := range rule.Ports {
			if port.Port == connection.Port && strings.EqualFold(port.Protocol, connection.Protocol) {
				portMatches = true
				break
			}
		}
	}

	if !portMatches {
		return false
	}

	// Check if source matches 'from' selectors
	if len(rule.From) == 0 {
		return true // No source restriction
	}

	for _, from := range rule.From {
		if sourcePod != nil && s.podMatchesSelector(sourcePod, from.PodSelector) {
			return true
		}
		// Additional checks for namespace selectors and IP blocks would go here
	}

	return false
}
