# Kubernetes Network Visualizer & Debugger

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21-blue.svg)](https://go.dev/)
[![React Version](https://img.shields.io/badge/React-18.2-blue.svg)](https://reactjs.org/)

## 🎯 Problem Statement

Networking issues in Azure Kubernetes Service (AKS) and other Kubernetes environments are notoriously difficult to debug:
- Connectivity drops between services
- Misconfigured services and endpoints
- Overlapping CIDRs causing routing issues
- Complex WireGuard/Konnectivity configurations
- Network policy conflicts
- DNS resolution failures

## 💡 Solution

A comprehensive tool that automatically visualizes and debugs Kubernetes network topology, providing:
- **Real-time network topology visualization** between pods, nodes, and external services
- **Automated health checks** with continuous connectivity monitoring
- **"What-if" simulations** to predict the impact of policy changes
- **Intelligent root cause analysis** for connectivity issues
- **Both Web UI and CLI interfaces** for different use cases

## 🏗️ Architecture

### Components

1. **Data Collector** (Go-based controller)
   - Queries Kubernetes API for network resources
   - Performs lightweight connectivity probes
   - Collects metrics and logs

2. **Analysis Engine**
   - Processes data into graph model
   - Detects and diagnoses network issues
   - Enriches failures with probable causes

3. **Visualization Layer**
   - Interactive web UI with D3.js/Cytoscape.js
   - Real-time updates via WebSocket
   - Detailed debug information panels

4. **CLI Tool**
   - Quick debugging without web UI
   - Scriptable for CI/CD pipelines
   - Export capabilities for reports

## 🚀 Quick Start

### Prerequisites
- Kubernetes cluster (AKS, EKS, GKE, or local)
- kubectl configured
- Helm 3.x (optional)
- Go 1.21+ (for development)
- Node.js 18+ (for frontend development)

### Installation

#### Using Helm
```bash
helm repo add k8s-net-viz https://christine33-creator.github.io/k8-network-visualizer
helm install network-visualizer k8s-net-viz/network-visualizer
```

#### Using kubectl
```bash
kubectl apply -f https://raw.githubusercontent.com/christine33-creator/k8-network-visualizer/main/deploy/manifests/all-in-one.yaml
```

#### CLI Installation
```bash
# Linux/macOS
curl -L https://github.com/christine33-creator/k8-network-visualizer/releases/latest/download/k8s-netvis-$(uname -s)-$(uname -m) -o k8s-netvis
chmod +x k8s-netvis
sudo mv k8s-netvis /usr/local/bin/

# Using Go
go install github.com/christine33-creator/k8-network-visualizer/cli/cmd@latest
```

## 📊 Features

### Network Topology Visualization
- **Interactive Graph**: Pan, zoom, and filter network connections
- **Color-coded Health Status**: Green (healthy), Yellow (degraded), Red (failed)
- **Hierarchical Layout**: Organize by namespace, node, or service

### Health Monitoring
- **Continuous Probing**: TCP, HTTP, gRPC health checks
- **Latency Tracking**: Response time measurements
- **Packet Loss Detection**: Identify unreliable connections

### Debugging Capabilities
- **Root Cause Analysis**: Automatic detection of common issues
- **Policy Validation**: Check NetworkPolicy conflicts
- **DNS Troubleshooting**: Resolve and validate DNS configurations
- **Firewall Detection**: Identify blocked traffic patterns

### What-If Simulations
- **Policy Impact Analysis**: Preview effects of NetworkPolicy changes
- **Service Mesh Integration**: Simulate Istio/Linkerd policy changes
- **Failure Scenarios**: Test resilience to node/pod failures

## 🛠️ Development

### Backend Development
```bash
cd backend
go mod init github.com/christine33-creator/k8-network-visualizer
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest
go run cmd/main.go
```

### Frontend Development
```bash
cd frontend
npm install
npm start
```

### CLI Development
```bash
cd cli
go build -o k8s-netvis cmd/main.go
./k8s-netvis --help
```

## 📝 Usage Examples

### Web UI
Access the web interface at `http://localhost:8080` after deployment.

### CLI Examples
```bash
# Visualize current network topology
k8s-netvis visualize --namespace default

# Run health checks
k8s-netvis health --all-namespaces

# Simulate NetworkPolicy changes
k8s-netvis simulate --policy new-policy.yaml

# Export topology as JSON
k8s-netvis export --format json --output network-topology.json
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🏆 Hackathon Information

This project is being developed for the Azure Kubernetes Service (AKS) Hackathon.

**Team**: Solo Developer
**Contact**: christine33-creator

## 🗺️ Roadmap

- [x] Initial project structure
- [ ] Core data collection service
- [ ] Graph analysis engine
- [ ] Basic web UI
- [ ] CLI tool
- [ ] Health check implementation
- [ ] What-if simulation engine
- [ ] Service mesh integration
- [ ] Multi-cluster support
- [ ] AI-powered root cause analysis
