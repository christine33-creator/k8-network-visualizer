# Network Traffic Flow Visualization - Setup Guide

This guide walks you through setting up and testing the Network Traffic Flow Visualization feature with Cilium/Hubble in minikube.

## Overview

The flow visualization feature provides:
- **Real-time network flow capture** via Cilium Hubble
- **Traffic animation** on the network graph
- **Anomaly detection** for unusual traffic patterns
- **Bandwidth visualization** with dynamic edge widths
- **WebSocket streaming** for live flow updates

## Architecture

```
┌─────────────────┐
│  Cilium Hubble  │  (eBPF-based flow collection)
│     Relay       │
└────────┬────────┘
         │ gRPC
         │
┌────────▼────────┐
│ Flow Collector  │  (backend/pkg/flowcollector)
│  + Anomaly      │
│   Detector      │
└────────┬────────┘
         │
┌────────▼────────┐
│  Graph Engine   │  (Updates edge flow data)
└────────┬────────┘
         │
┌────────▼────────┐
│  WebSocket      │  (Real-time streaming)
└────────┬────────┘
         │
┌────────▼────────┐
│  React UI       │  (Animated graph visualization)
└─────────────────┘
```

## Prerequisites

- **minikube** v1.30+
- **kubectl** v1.28+
- **Docker** (for building images)
- **Helm** v3.0+ (for installing Cilium)
- At least 4GB RAM allocated to minikube

## Step 1: Start Minikube with Cilium CNI

```bash
# Delete existing minikube cluster if needed
minikube delete

# Start minikube with Cilium CNI
minikube start \
  --network-plugin=cni \
  --cni=false \
  --memory=6144 \
  --cpus=4 \
  --driver=docker

# Verify minikube is running
minikube status
```

## Step 2: Install Cilium with Hubble

```bash
# Add Cilium Helm repository
helm repo add cilium https://helm.cilium.io/
helm repo update

# Install Cilium with Hubble enabled
helm install cilium cilium/cilium \
  --version 1.14.0 \
  --namespace kube-system \
  --set hubble.relay.enabled=true \
  --set hubble.ui.enabled=true \
  --set hubble.metrics.enabled="{dns,drop,tcp,flow,icmp,http}"

# Wait for Cilium to be ready
kubectl -n kube-system rollout status deployment/cilium-operator
kubectl -n kube-system rollout status daemonset/cilium

# Verify Cilium status
kubectl -n kube-system exec -it ds/cilium -- cilium status
```

Expected output should show:
```
Cilium:         OK
Hubble:         OK
ClusterMesh:    Disabled
```

## Step 3: Verify Hubble Relay

```bash
# Check Hubble relay is running
kubectl -n kube-system get pods -l k8s-app=hubble-relay

# Port-forward Hubble relay for testing
kubectl -n kube-system port-forward svc/hubble-relay 4245:80 &

# Test Hubble connectivity (requires hubble CLI - optional)
# hubble status --server localhost:4245
```

## Step 4: Build and Deploy the Network Visualizer

```bash
# Build the backend
cd backend
go mod tidy
go build -o network-visualizer ./cmd

# Build Docker image for minikube
eval $(minikube docker-env)
docker build -t k8s-network-visualizer:latest -f ../Dockerfile ..

# Build frontend
cd ../frontend
npm install
npm run build

# Deploy to minikube
cd ..
kubectl apply -f deploy/manifests/deployment.yaml

# Wait for deployment
kubectl -n network-visualizer rollout status deployment/network-visualizer
```

## Step 5: Enable Flow Collection

Update the deployment to enable flows:

```bash
kubectl -n network-visualizer set env deployment/network-visualizer \
  ENABLE_FLOWS=true \
  HUBBLE_ADDR=hubble-relay.kube-system.svc.cluster.local:80
```

## Step 6: Deploy Test Applications

```bash
# Deploy test workloads
kubectl apply -f deploy/test-apps.yaml

# Verify test pods are running
kubectl get pods -n test-apps
```

## Step 7: Generate Traffic for Testing

```bash
# Apply the traffic generator script
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: traffic-generator
  namespace: test-apps
spec:
  containers:
  - name: traffic-gen
    image: nicolaka/netshoot
    command: ["/bin/bash"]
    args:
      - -c
      - |
        while true; do
          # HTTP traffic to backend
          curl -s http://backend-api-service:8080/health || true
          
          # DNS lookups
          nslookup backend-api-service || true
          nslookup frontend-service || true
          
          # TCP connections
          nc -zv backend-api-service 8080 || true
          
          sleep 2
        done
EOF

# Watch traffic generator logs
kubectl -n test-apps logs -f traffic-generator
```

## Step 8: Access the Visualization

```bash
# Port-forward the network visualizer UI
kubectl -n network-visualizer port-forward svc/network-visualizer 8080:80

# Open in browser
echo "Open http://localhost:8080 in your browser"
```

## Step 9: Verify Flow Visualization

In the browser at http://localhost:8080:

1. **Check Active Flows Badge**: Should show active flow count in the top bar
2. **Observe Animated Edges**: Edges should pulse/flash blue when traffic flows
3. **Check Edge Width**: Edges with more traffic should be thicker
4. **View Flow Metrics**: Click on an edge to see:
   - Bytes/sec bandwidth
   - Packets/sec
   - Connection count
   - Error rate
   - Protocol
5. **Monitor Anomalies**: Orange warning chip shows detected anomalies

## API Endpoints for Testing

### Get Recent Flows
```bash
curl http://localhost:8080/api/flows?limit=10 | jq
```

### Get Flow Metrics
```bash
curl http://localhost:8080/api/flows/metrics | jq
```

### Get Flow Anomalies
```bash
curl http://localhost:8080/api/flows/anomalies | jq
```

### Get Active Flows
```bash
curl http://localhost:8080/api/flows/active | jq
```

## Testing Scenarios

### 1. Traffic Spike Detection

```bash
# Generate high traffic
kubectl -n test-apps exec traffic-generator -- bash -c '
for i in {1..100}; do
  curl -s http://backend-api-service:8080/health &
done
wait
'

# Check for anomalies
curl http://localhost:8080/api/flows/anomalies?severity=high | jq
```

### 2. Port Scanning Detection

```bash
# Simulate port scan
kubectl -n test-apps exec traffic-generator -- bash -c '
for port in {1..30}; do
  nc -zv -w 1 backend-api-service $port 2>&1
done
'

# Check for port scan anomaly
curl http://localhost:8080/api/flows/anomalies | jq '.[] | select(.type=="port_scan")'
```

### 3. Error Rate Detection

```bash
# Generate errors by connecting to non-existent service
kubectl -n test-apps exec traffic-generator -- bash -c '
for i in {1..20}; do
  curl -s http://non-existent-service:8080 || true
done
'

# Check for high error rate
curl http://localhost:8080/api/flows/anomalies | jq '.[] | select(.type=="high_error_rate")'
```

## Troubleshooting

### Flows not appearing

```bash
# Check Hubble relay logs
kubectl -n kube-system logs deployment/hubble-relay

# Check network visualizer logs
kubectl -n network-visualizer logs deployment/network-visualizer

# Verify Hubble is collecting flows
kubectl -n kube-system exec -it ds/cilium -- cilium hubble observe --last 10
```

### WebSocket connection failing

```bash
# Check WebSocket endpoint
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
  http://localhost:8080/ws/flows
```

### No anomalies detected

```bash
# Check if baseline is being built
# Anomaly detection requires at least 10 samples for baseline
# Wait 2-3 minutes after deployment for baselines to establish

# Check detector state via logs
kubectl -n network-visualizer logs deployment/network-visualizer | grep -i anomaly
```

## Performance Tuning

### Adjust flow retention

Edit `backend/pkg/flowcollector/flowcollector.go`:
```go
maxFlows:     10000, // Increase for more history
metricWindow: 60 * time.Second, // Adjust aggregation window
```

### Adjust anomaly thresholds

Edit `backend/pkg/flowcollector/anomaly.go`:
```go
spikeThreshold:      3.0,  // 3x baseline = spike
errorRateThreshold:  0.05, // 5% error rate
portScanThreshold:   20,   // 20 unique ports
exfilThreshold:      10 * 1024 * 1024, // 10 MB/s
```

## Cleanup

```bash
# Delete test traffic generator
kubectl -n test-apps delete pod traffic-generator

# Delete test apps
kubectl delete -f deploy/test-apps.yaml

# Delete network visualizer
kubectl delete -f deploy/manifests/deployment.yaml

# Uninstall Cilium (optional)
helm uninstall cilium -n kube-system

# Delete minikube cluster
minikube delete
```

## Next Steps

- Integrate with Prometheus for historical flow metrics
- Add flow filtering by namespace/protocol in UI
- Implement flow path tracing for multi-hop connections
- Add export functionality for flow data
- Create alerts based on anomaly thresholds

## References

- [Cilium Documentation](https://docs.cilium.io/)
- [Hubble Documentation](https://docs.cilium.io/en/stable/observability/hubble/)
- [eBPF Introduction](https://ebpf.io/)
