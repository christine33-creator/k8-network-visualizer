package simulation

import (
	"testing"

	"github.com/christine33-creator/k8-network-visualizer/pkg/models"
)

func TestSimulatePortBlocking(t *testing.T) {
	simulator := NewSimulator()

	topology := &models.NetworkTopology{
		Connections: []models.Connection{
			{
				Source:      "10.0.1.1",
				Destination: "10.0.1.2",
				Port:        80,
				Protocol:    "TCP",
				Status:      "active",
			},
			{
				Source:      "10.0.1.1",
				Destination: "10.0.1.3",
				Port:        443,
				Protocol:    "TCP",
				Status:      "active",
			},
		},
	}

	simulation := simulator.SimulatePortBlocking(topology, 80, "TCP")

	if simulation == nil {
		t.Fatal("Expected simulation result, got nil")
	}

	if len(simulation.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(simulation.Results))
	}

	// Check that port 80 connection is blocked
	for _, result := range simulation.Results {
		if result.Connection.Port == 80 {
			if result.Impact != "blocked" {
				t.Errorf("Expected port 80 connection to be blocked, got %s", result.Impact)
			}
		} else {
			if result.Impact != "no_change" {
				t.Errorf("Expected non-port 80 connection to have no change, got %s", result.Impact)
			}
		}
	}
}

func TestSimulatePodFailure(t *testing.T) {
	simulator := NewSimulator()

	topology := &models.NetworkTopology{
		Pods: []models.Pod{
			{
				Name:      "test-pod",
				Namespace: "default",
				IP:        "10.0.1.100",
			},
		},
		Connections: []models.Connection{
			{
				Source:      "10.0.1.100",
				Destination: "10.0.1.2",
				Port:        80,
				Protocol:    "TCP",
				Status:      "active",
			},
			{
				Source:      "10.0.1.1",
				Destination: "10.0.1.100",
				Port:        443,
				Protocol:    "TCP",
				Status:      "active",
			},
			{
				Source:      "10.0.1.1",
				Destination: "10.0.1.3",
				Port:        22,
				Protocol:    "TCP",
				Status:      "active",
			},
		},
	}

	simulation := simulator.SimulatePodFailure(topology, "test-pod", "default")

	if simulation == nil {
		t.Fatal("Expected simulation result, got nil")
	}

	if len(simulation.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(simulation.Results))
	}

	// Check that connections involving the failed pod are blocked
	blockedCount := 0
	for _, result := range simulation.Results {
		if result.Connection.Source == "10.0.1.100" || result.Connection.Destination == "10.0.1.100" {
			if result.Impact != "blocked" {
				t.Errorf("Expected connection involving failed pod to be blocked, got %s", result.Impact)
			}
			blockedCount++
		}
	}

	if blockedCount != 2 {
		t.Errorf("Expected 2 connections to be blocked, got %d", blockedCount)
	}
}

func TestPodMatchesSelector(t *testing.T) {
	simulator := NewSimulator()

	pod := &models.Pod{
		Labels: map[string]string{
			"app":     "web",
			"version": "v1",
		},
	}

	tests := []struct {
		name     string
		selector map[string]string
		expected bool
	}{
		{
			name:     "matches all labels",
			selector: map[string]string{"app": "web", "version": "v1"},
			expected: true,
		},
		{
			name:     "matches subset",
			selector: map[string]string{"app": "web"},
			expected: true,
		},
		{
			name:     "no match",
			selector: map[string]string{"app": "api"},
			expected: false,
		},
		{
			name:     "empty selector",
			selector: map[string]string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simulator.podMatchesSelector(pod, tt.selector)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
