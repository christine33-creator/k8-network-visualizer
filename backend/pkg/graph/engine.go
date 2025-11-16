package graph

import (
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// NodeType represents the type of node in the graph
type NodeType string

const (
	NodeTypePod         NodeType = "pod"
	NodeTypeService     NodeType = "service"
	NodeTypeNode        NodeType = "node"
	NodeTypeNamespace   NodeType = "namespace"
	NodeTypeExternal    NodeType = "external"
)

// EdgeType represents the type of edge in the graph
type EdgeType string

const (
	EdgeTypeConnection  EdgeType = "connection"
	EdgeTypeService     EdgeType = "service"
	EdgeTypePolicy      EdgeType = "policy"
)

// GraphNode represents a node in the network graph
type GraphNode struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       NodeType          `json:"type"`
	Namespace  string            `json:"namespace,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Health     HealthStatus      `json:"health"`
	PodIP      string            `json:"pod_ip,omitempty"`
	NodeName   string            `json:"node_name,omitempty"`
}

// GraphEdge represents an edge in the network graph
type GraphEdge struct {
	ID         string            `json:"id"`
	Source     string            `json:"source"`
	Target     string            `json:"target"`
	Type       EdgeType          `json:"type"`
	Properties map[string]string `json:"properties,omitempty"`
	Health     HealthStatus      `json:"health"`
	Latency    int64             `json:"latency_ms,omitempty"`
	PacketLoss float64           `json:"packet_loss,omitempty"`
	// Flow metrics
	FlowData   *FlowData         `json:"flow_data,omitempty"`
}

// FlowData represents real-time network flow information
type FlowData struct {
	BytesPerSec     float64 `json:"bytes_per_sec"`
	PacketsPerSec   float64 `json:"packets_per_sec"`
	ConnectionCount int64   `json:"connection_count"`
	ErrorRate       float64 `json:"error_rate"`
	Protocol        string  `json:"protocol"`
	LastSeen        string  `json:"last_seen"`
	IsActive        bool    `json:"is_active"`
	Direction       string  `json:"direction"` // bidirectional, ingress, egress
}

// HealthStatus represents the health of a node or edge
type HealthStatus string

const (
	HealthHealthy  HealthStatus = "healthy"
	HealthDegraded HealthStatus = "degraded"
	HealthFailed   HealthStatus = "failed"
	HealthUnknown  HealthStatus = "unknown"
)

// NetworkTopology represents the complete network topology
type NetworkTopology struct {
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	Timestamp time.Time   `json:"timestamp"`
}

// Engine manages the network graph
type Engine struct {
	mu       sync.RWMutex
	nodes    map[string]*GraphNode
	edges    map[string]*GraphEdge
	topology *NetworkTopology
}

// NewEngine creates a new graph engine
func NewEngine() *Engine {
	return &Engine{
		nodes: make(map[string]*GraphNode),
		edges: make(map[string]*GraphEdge),
		topology: &NetworkTopology{
			Nodes:     []GraphNode{},
			Edges:     []GraphEdge{},
			Timestamp: time.Now(),
		},
	}
}

// AddPod adds a pod to the graph
func (e *Engine) AddPod(pod *corev1.Pod) {
	e.mu.Lock()
	defer e.mu.Unlock()

	nodeID := fmt.Sprintf("pod/%s/%s", pod.Namespace, pod.Name)
	node := &GraphNode{
		ID:        nodeID,
		Name:      pod.Name,
		Type:      NodeTypePod,
		Namespace: pod.Namespace,
		Labels:    pod.Labels,
		Properties: map[string]string{
			"status": string(pod.Status.Phase),
		},
		PodIP:    pod.Status.PodIP,
		NodeName: pod.Spec.NodeName,
	}

	// Set health status based on pod phase
	switch pod.Status.Phase {
	case corev1.PodRunning:
		node.Health = HealthHealthy
	case corev1.PodPending:
		node.Health = HealthDegraded
	case corev1.PodFailed:
		node.Health = HealthFailed
	default:
		node.Health = HealthUnknown
	}

	e.nodes[nodeID] = node
}

// AddService adds a service to the graph
func (e *Engine) AddService(svc *corev1.Service) {
	e.mu.Lock()
	defer e.mu.Unlock()

	nodeID := fmt.Sprintf("service/%s/%s", svc.Namespace, svc.Name)
	node := &GraphNode{
		ID:        nodeID,
		Name:      svc.Name,
		Type:      NodeTypeService,
		Namespace: svc.Namespace,
		Labels:    svc.Labels,
		Properties: map[string]string{
			"type":       string(svc.Spec.Type),
			"cluster_ip": svc.Spec.ClusterIP,
		},
		Health: HealthHealthy, // Services are considered healthy by default
	}

	e.nodes[nodeID] = node
}

// AddNode adds a Kubernetes node to the graph
func (e *Engine) AddNode(node *corev1.Node) {
	e.mu.Lock()
	defer e.mu.Unlock()

	nodeID := fmt.Sprintf("node/%s", node.Name)
	graphNode := &GraphNode{
		ID:     nodeID,
		Name:   node.Name,
		Type:   NodeTypeNode,
		Labels: node.Labels,
		Properties: map[string]string{
			"provider_id": node.Spec.ProviderID,
		},
	}

	// Check node conditions for health
	graphNode.Health = HealthHealthy
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status != corev1.ConditionTrue {
				graphNode.Health = HealthFailed
			}
		}
	}

	e.nodes[nodeID] = graphNode
}

// AddServiceEndpoint creates edges between services and pods
func (e *Engine) AddServiceEndpoint(svc *corev1.Service, endpoints *corev1.Endpoints) {
	e.mu.Lock()
	defer e.mu.Unlock()

	serviceID := fmt.Sprintf("service/%s/%s", svc.Namespace, svc.Name)

	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			if address.TargetRef != nil && address.TargetRef.Kind == "Pod" {
				podID := fmt.Sprintf("pod/%s/%s", address.TargetRef.Namespace, address.TargetRef.Name)
				edgeID := fmt.Sprintf("%s->%s", serviceID, podID)

				edge := &GraphEdge{
					ID:     edgeID,
					Source: serviceID,
					Target: podID,
					Type:   EdgeTypeService,
					Properties: map[string]string{
						"ip": address.IP,
					},
					Health: HealthHealthy,
				}

				e.edges[edgeID] = edge
			}
		}
	}
}

// AddConnection adds a connection edge between two nodes
func (e *Engine) AddConnection(sourceID, targetID string, latency int64, success bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	edgeID := fmt.Sprintf("%s->%s", sourceID, targetID)

	health := HealthHealthy
	if !success {
		health = HealthFailed
	} else if latency > 100 {
		health = HealthDegraded
	}

	edge := &GraphEdge{
		ID:      edgeID,
		Source:  sourceID,
		Target:  targetID,
		Type:    EdgeTypeConnection,
		Health:  health,
		Latency: latency,
	}

	e.edges[edgeID] = edge
}

// AddNetworkPolicy adds network policy relationships to the graph
func (e *Engine) AddNetworkPolicy(policy *networkingv1.NetworkPolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Create edges for pods selected by the policy
	// This is simplified - in production, we'd need to evaluate the selectors
	policyID := fmt.Sprintf("policy/%s/%s", policy.Namespace, policy.Name)
	
	// Add policy as a node
	node := &GraphNode{
		ID:        policyID,
		Name:      policy.Name,
		Type:      NodeTypeNamespace, // Using namespace type for policies for now
		Namespace: policy.Namespace,
		Properties: map[string]string{
			"type": "NetworkPolicy",
		},
		Health: HealthHealthy,
	}
	e.nodes[policyID] = node
}

// GetTopology returns the current network topology
func (e *Engine) GetTopology() *NetworkTopology {
	e.mu.RLock()
	defer e.mu.RUnlock()

	nodes := make([]GraphNode, 0, len(e.nodes))
	for _, node := range e.nodes {
		nodes = append(nodes, *node)
	}

	edges := make([]GraphEdge, 0, len(e.edges))
	for _, edge := range e.edges {
		edges = append(edges, *edge)
	}

	return &NetworkTopology{
		Nodes:     nodes,
		Edges:     edges,
		Timestamp: time.Now(),
	}
}

// GetNodeByID returns a node by its ID
func (e *Engine) GetNodeByID(id string) (*GraphNode, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	node, exists := e.nodes[id]
	if !exists {
		return nil, false
	}

	nodeCopy := *node
	return &nodeCopy, true
}

// GetEdgesBySource returns all edges originating from a source node
func (e *Engine) GetEdgesBySource(sourceID string) []GraphEdge {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var edges []GraphEdge
	for _, edge := range e.edges {
		if edge.Source == sourceID {
			edges = append(edges, *edge)
		}
	}

	return edges
}

// GetEdgesByTarget returns all edges targeting a node
func (e *Engine) GetEdgesByTarget(targetID string) []GraphEdge {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var edges []GraphEdge
	for _, edge := range e.edges {
		if edge.Target == targetID {
			edges = append(edges, *edge)
		}
	}

	return edges
}

// UpdateNodeHealth updates the health status of a node
func (e *Engine) UpdateNodeHealth(nodeID string, health HealthStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if node, exists := e.nodes[nodeID]; exists {
		node.Health = health
	}
}

// UpdateEdgeHealth updates the health status of an edge
func (e *Engine) UpdateEdgeHealth(edgeID string, health HealthStatus, latency int64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if edge, exists := e.edges[edgeID]; exists {
		edge.Health = health
		edge.Latency = latency
	}
}

// UpdateEdgeFlowData updates flow data for an edge
func (e *Engine) UpdateEdgeFlowData(sourceID, targetID string, flowData *FlowData) {
	e.mu.Lock()
	defer e.mu.Unlock()

	edgeID := fmt.Sprintf("%s->%s", sourceID, targetID)
	edge, exists := e.edges[edgeID]
	if !exists {
		// Create edge if it doesn't exist
		edge = &GraphEdge{
			ID:     edgeID,
			Source: sourceID,
			Target: targetID,
			Type:   EdgeTypeConnection,
			Health: HealthHealthy,
		}
		e.edges[edgeID] = edge
	}

	edge.FlowData = flowData
	
	// Update health based on flow metrics
	if flowData.ErrorRate > 0.1 {
		edge.Health = HealthFailed
	} else if flowData.ErrorRate > 0.05 {
		edge.Health = HealthDegraded
	} else if flowData.IsActive {
		edge.Health = HealthHealthy
	}
}

// GetActiveFlows returns edges with active flow data
func (e *Engine) GetActiveFlows() []GraphEdge {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var activeFlows []GraphEdge
	for _, edge := range e.edges {
		if edge.FlowData != nil && edge.FlowData.IsActive {
			activeFlows = append(activeFlows, *edge)
		}
	}

	return activeFlows
}

// Clear removes all nodes and edges from the graph
func (e *Engine) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.nodes = make(map[string]*GraphNode)
	e.edges = make(map[string]*GraphEdge)
}