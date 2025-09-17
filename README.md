# Kubernetes Network Visualizer

A tool that automatically visualizes network topology between pods, nodes, and external services in Kubernetes clusters, with health checks and "what if" simulations to help debug networking issues in AKS and other Kubernetes environments.

## Features

- **Network Topology Discovery**: Automatically discovers and maps relationships between nodes, pods, services, and network policies
- **Multiple Output Formats**: CLI text output, JSON export, and interactive web visualization
- **Health Checks**: Tests connectivity between network components
- **What-If Simulations**: Simulate the effects of network policy changes, port blocking, and pod failures
- **AKS Optimized**: Specifically designed to help debug common AKS networking issues like:
  - Connectivity drops
  - Misconfigured services
  - Overlapping CIDRs
  - WireGuard/Konnectivity complexity

## Installation

### Build from Source

```bash
git clone https://github.com/christine33-creator/k8-network-visualizer.git
cd k8-network-visualizer
go build -o k8-network-visualizer cmd/main.go
```

### Prerequisites

- Go 1.19 or later
- Access to a Kubernetes cluster
- kubectl configured with appropriate permissions

## Usage

### Basic Usage

```bash
# Analyze current cluster with CLI output
./k8-network-visualizer

# Analyze specific namespace
./k8-network-visualizer -namespace=production

# Generate JSON output
./k8-network-visualizer -output=json > topology.json

# Start web interface
./k8-network-visualizer -output=web -port=8080
```

### Command Line Options

```
-kubeconfig string
    Path to kubeconfig file (uses in-cluster config if empty)
-namespace string
    Kubernetes namespace to analyze (empty for all namespaces)
-output string
    Output format: cli, json, web (default "cli")
-port int
    Port for web interface (default 8080)
-verbose
    Enable verbose logging
```

### Web Interface

The web interface provides an interactive visualization of your network topology:

```bash
./k8-network-visualizer -output=web
# Open http://localhost:8080 in your browser
```

Features:
- Interactive network graph with D3.js
- Drag and drop nodes
- Color-coded components (nodes, pods, services)
- Detailed component information cards
- Real-time topology statistics

## Examples

### Analyzing a Specific Namespace

```bash
./k8-network-visualizer -namespace=kube-system -verbose
```

### Export Network Topology

```bash
./k8-network-visualizer -output=json | jq '.'
```

### Using with Different Kubeconfig

```bash
./k8-network-visualizer -kubeconfig=/path/to/custom/kubeconfig
```

## Architecture

The tool is structured into several key components:

- **CLI Interface** (`internal/cli`): Command-line interface and output formatting
- **Kubernetes Client** (`internal/k8s`): Kubernetes API interaction and resource discovery
- **Visualizer** (`internal/visualizer`): Web interface and HTML generation
- **Health Checker** (`internal/health`): Network connectivity testing
- **Simulator** (`internal/simulation`): "What if" scenario analysis
- **Models** (`pkg/models`): Data structures for network topology

## API Reference

### Network Topology Structure

The tool discovers and represents the following components:

```go
type NetworkTopology struct {
    Nodes       []Node          `json:"nodes"`
    Pods        []Pod           `json:"pods"`
    Services    []Service       `json:"services"`
    Connections []Connection    `json:"connections"`
    Policies    []NetworkPolicy `json:"policies"`
    Timestamp   time.Time       `json:"timestamp"`
}
```

### Health Checks

```go
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
```

### Simulations

```go
type Simulation struct {
    Name        string             `json:"name"`
    Description string             `json:"description"`
    Changes     []SimulationChange `json:"changes"`
    Results     []SimulationResult `json:"results"`
}
```

## Troubleshooting Common AKS Issues

### Connectivity Drops
1. Use the health checker to identify failing connections
2. Check network policies that might be blocking traffic
3. Verify service endpoints are healthy

### Misconfigured Services
1. Review service-to-pod mappings in the topology
2. Check selector labels match pod labels
3. Verify port configurations

### Overlapping CIDRs
1. Examine node CIDR allocations in the nodes view
2. Look for IP conflicts in the topology

### WireGuard/Konnectivity Issues
1. Check node-to-node connectivity
2. Verify cluster networking policies
3. Use simulations to test policy changes

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Roadmap

- [ ] Real-time health monitoring dashboard
- [ ] Network policy recommendation engine
- [ ] Integration with popular service meshes (Istio, Linkerd)
- [ ] Support for CNI-specific features
- [ ] Alerting and notification system
- [ ] Performance metrics collection
- [ ] Custom visualization plugins
