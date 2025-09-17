package models

import (
	"time"
)

// NetworkTopology represents the complete network topology
type NetworkTopology struct {
	Nodes       []Node          `json:"nodes"`
	Pods        []Pod           `json:"pods"`
	Services    []Service       `json:"services"`
	Connections []Connection    `json:"connections"`
	Policies    []NetworkPolicy `json:"policies"`
	Timestamp   time.Time       `json:"timestamp"`
}

// Node represents a Kubernetes node
type Node struct {
	Name   string            `json:"name"`
	IP     string            `json:"ip"`
	Labels map[string]string `json:"labels"`
	Ready  bool              `json:"ready"`
	CIDRs  []string          `json:"cidrs"`
	Pods   []string          `json:"pods"` // Pod names running on this node
}

// Pod represents a Kubernetes pod
type Pod struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	IP        string            `json:"ip"`
	Node      string            `json:"node"`
	Labels    map[string]string `json:"labels"`
	Status    string            `json:"status"`
	Ports     []Port            `json:"ports"`
}

// Service represents a Kubernetes service
type Service struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	ClusterIP string            `json:"cluster_ip"`
	Ports     []Port            `json:"ports"`
	Selector  map[string]string `json:"selector"`
	Endpoints []string          `json:"endpoints"` // Pod IPs
}

// Port represents a network port
type Port struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

// Connection represents a network connection between components
type Connection struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	Status      string `json:"status"` // "active", "blocked", "unknown"
}

// NetworkPolicy represents a Kubernetes network policy
type NetworkPolicy struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ingress   []Rule            `json:"ingress"`
	Egress    []Rule            `json:"egress"`
	Selector  map[string]string `json:"selector"`
}

// Rule represents a network policy rule
type Rule struct {
	From  []Selector `json:"from"`
	To    []Selector `json:"to"`
	Ports []Port     `json:"ports"`
}

// Selector represents a network policy selector
type Selector struct {
	PodSelector       map[string]string `json:"pod_selector,omitempty"`
	NamespaceSelector map[string]string `json:"namespace_selector,omitempty"`
	IPBlock           *IPBlock          `json:"ip_block,omitempty"`
}

// IPBlock represents an IP block in network policy
type IPBlock struct {
	CIDR   string   `json:"cidr"`
	Except []string `json:"except,omitempty"`
}

// HealthCheck represents a network health check result
type HealthCheck struct {
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	Port        int32         `json:"port"`
	Protocol    string        `json:"protocol"`
	Status      string        `json:"status"` // "success", "failed", "timeout"
	Latency     time.Duration `json:"latency,omitempty"`
	Error       string        `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// Simulation represents a "what if" simulation scenario
type Simulation struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Changes     []SimulationChange `json:"changes"`
	Results     []SimulationResult `json:"results"`
}

// SimulationChange represents a change in a simulation
type SimulationChange struct {
	Type   string      `json:"type"` // "add_policy", "remove_policy", "block_port", etc.
	Target string      `json:"target"`
	Data   interface{} `json:"data"`
}

// SimulationResult represents the result of a simulation
type SimulationResult struct {
	Connection Connection `json:"connection"`
	Before     string     `json:"before"` // status before change
	After      string     `json:"after"`  // status after change
	Impact     string     `json:"impact"` // "blocked", "allowed", "no_change"
}
