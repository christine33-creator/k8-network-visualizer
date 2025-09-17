# 🚀 Quick Start Guide - Kubernetes Network Visualizer

## Prerequisites

- Kubernetes cluster (AKS, EKS, GKE, or local)
- kubectl configured and connected to your cluster
- Docker (for building images)
- Go 1.21+ and Node.js 18+ (for local development)

## 🎯 Quick Deploy to Kubernetes

### 1. Deploy from Pre-built Image

```bash
# Apply the deployment manifest
kubectl apply -f https://raw.githubusercontent.com/christine33-creator/k8-network-visualizer/main/deploy/manifests/deployment.yaml

# Wait for the pod to be ready
kubectl wait --for=condition=ready pod -l app=network-visualizer -n network-visualizer --timeout=120s

# Get the service URL (LoadBalancer IP)
kubectl get service network-visualizer -n network-visualizer

# Port-forward for immediate access (if LoadBalancer is pending)
kubectl port-forward -n network-visualizer service/network-visualizer 8080:80
```

Access the UI at: http://localhost:8080

### 2. Deploy to AKS with Application Gateway

```bash
# Ensure you have Application Gateway Ingress Controller installed
# Update the ingress host in deployment.yaml with your domain

kubectl apply -f deploy/manifests/deployment.yaml

# Check ingress status
kubectl get ingress -n network-visualizer
```

## 💻 Local Development

### Backend

```bash
cd backend
go mod download
go run cmd/main.go --kubeconfig=$HOME/.kube/config
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Access at: http://localhost:5173

### CLI

```bash
cd cli
go build -o k8s-netvis cmd/main.go
./k8s-netvis --server http://localhost:8080 visualize
```

## 🔧 Using the CLI Tool

### Install CLI

```bash
# Build from source
cd cli && go build -o k8s-netvis cmd/main.go
sudo mv k8s-netvis /usr/local/bin/

# Or download pre-built (when available)
curl -L https://github.com/christine33-creator/k8-network-visualizer/releases/latest/download/k8s-netvis -o k8s-netvis
chmod +x k8s-netvis
sudo mv k8s-netvis /usr/local/bin/
```

### CLI Commands

```bash
# View network topology
k8s-netvis visualize --namespace default

# Check cluster health
k8s-netvis health --all-namespaces

# List network issues
k8s-netvis issues --severity critical

# Run connectivity probe
k8s-netvis probe --source default/frontend-pod --target default/backend-service

# Export topology as JSON
k8s-netvis export --format json --output topology.json

# Simulate NetworkPolicy impact
k8s-netvis simulate --policy new-policy.yaml
```

## 🐳 Building Docker Image

```bash
# Build the image
docker build -t k8s-network-visualizer:latest .

# Tag for your registry
docker tag k8s-network-visualizer:latest <your-registry>/k8s-network-visualizer:latest

# Push to registry
docker push <your-registry>/k8s-network-visualizer:latest

# Update deployment.yaml with your image
kubectl set image deployment/network-visualizer backend=<your-registry>/k8s-network-visualizer:latest -n network-visualizer
```

## 📊 Features Overview

### Web UI
- **Interactive Graph**: Real-time network topology visualization
- **Health Monitoring**: Color-coded node and edge health status
- **Issue Detection**: Automatic detection of network problems
- **Detail Views**: Click nodes/edges for detailed information

### CLI Tool
- **Quick Diagnostics**: Run network health checks from terminal
- **Export Capabilities**: Export topology and issues as JSON
- **Scriptable**: Integrate into CI/CD pipelines
- **Policy Simulation**: Test NetworkPolicy changes before applying

### Backend API
- **REST Endpoints**: 
  - GET `/api/topology` - Network topology
  - GET `/api/health` - Health status
  - GET `/api/issues` - Detected issues
  - GET `/api/probes` - Probe results
  - POST `/api/simulate` - Policy simulation

## 🔍 Common Use Cases

### 1. Debug Service Connectivity
```bash
# Check if pods can reach a service
k8s-netvis probe --source namespace/pod-name --target namespace/service-name

# View all connectivity issues
k8s-netvis issues --type connectivity
```

### 2. Network Policy Validation
```bash
# Before applying a new policy
k8s-netvis simulate --policy network-policy.yaml

# Check for pods without policies
k8s-netvis issues --type policy
```

### 3. Performance Monitoring
```bash
# Check for high latency connections
k8s-netvis issues --type latency

# Export topology for analysis
k8s-netvis export --format json --output network-$(date +%Y%m%d).json
```

## 🆘 Troubleshooting

### Pod Not Starting
```bash
# Check logs
kubectl logs -n network-visualizer deployment/network-visualizer

# Check RBAC permissions
kubectl auth can-i get pods --as=system:serviceaccount:network-visualizer:network-visualizer
```

### Cannot Access UI
```bash
# Use port-forward for testing
kubectl port-forward -n network-visualizer service/network-visualizer 8080:80

# Check service endpoints
kubectl get endpoints -n network-visualizer
```

### CLI Connection Issues
```bash
# Verify server is running
curl http://localhost:8080/api/health

# Check network connectivity
k8s-netvis --server http://<service-ip> health
```

## 📚 Additional Resources

- [Full Documentation](README.md)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api.md)
- [Contributing Guide](docs/contributing.md)

## 🏆 Hackathon Demo

For the AKS hackathon demo:

1. Deploy to AKS cluster
2. Generate some network traffic between services
3. Open the Web UI to show real-time visualization
4. Use CLI to demonstrate issue detection
5. Simulate a NetworkPolicy change
6. Show how the tool helps debug connectivity issues

Good luck with your hackathon! 🎉