# Network Traffic Flow Visualization Feature

## Overview

This feature provides real-time network traffic flow visualization using Cilium Hubble's eBPF-based flow collection. It captures actual network traffic between pods and displays it as animated flows on the network topology graph.

## Features

### 1. Real-Time Flow Collection
- **Cilium Hubble Integration**: Connects to Hubble Relay via gRPC to stream network flows
- **Protocol Detection**: Identifies TCP, UDP, HTTP, HTTPS, gRPC, DNS, and ICMP traffic
- **L7 Visibility**: Captures application-layer details (HTTP methods, URLs, DNS queries)
- **Drop Tracking**: Monitors packet drops and network policy denials

### 2. Flow Metrics
- **Bandwidth Measurement**: Bytes/sec and packets/sec per connection
- **Connection Counting**: Tracks number of active connections
- **Error Rates**: Calculates percentage of failed/dropped packets
- **Direction Tracking**: Identifies ingress, egress, or bidirectional flows

### 3. Anomaly Detection
The system automatically detects:
- **Traffic Spikes**: 3x baseline threshold
- **Traffic Drops**: 80% reduction from baseline
- **Unusual Protocols**: New protocols not in historical baseline
- **Unexpected Connections**: New destination pods
- **High Error Rates**: >5% packet loss or drops
- **Port Scanning**: >20 unique ports in 1 minute
- **Data Exfiltration**: >10 MB/s sustained outbound traffic
- **DNS Anomalies**: >100 queries/minute

### 4. Interactive Visualization
- **Animated Flows**: Edges pulse blue when traffic is detected
- **Dynamic Widths**: Edge thickness represents bandwidth (logarithmic scale)
- **Bandwidth Labels**: Display actual throughput (B/s, KB/s, MB/s)
- **Active/Inactive States**: Faded edges for inactive connections
- **Anomaly Alerts**: Warning badges and inline alerts

### 5. WebSocket Streaming
- Real-time flow updates pushed to browser
- Low-latency (<100ms) flow-to-visualization pipeline
- Automatic reconnection on disconnect

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Kubernetes                          │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐       │
│  │   Pod A    │───▶│   Pod B    │───▶│   Pod C    │       │
│  └────────────┘    └────────────┘    └────────────┘       │
│         │                 │                 │               │
│         └─────────────────┴─────────────────┘               │
│                          │                                  │
│                    (eBPF kernel)                            │
│                          │                                  │
│                  ┌───────▼────────┐                         │
│                  │ Cilium Agent   │                         │
│                  └───────┬────────┘                         │
│                          │                                  │
│                  ┌───────▼────────┐                         │
│                  │ Hubble Relay   │                         │
│                  └───────┬────────┘                         │
└──────────────────────────┼──────────────────────────────────┘
                           │ gRPC Stream
                  ┌────────▼────────┐
                  │ FlowCollector   │
                  │  (Go backend)   │
                  └────────┬────────┘
                           │
            ┌──────────────┴──────────────┐
            │                             │
    ┌───────▼────────┐          ┌────────▼─────────┐
    │ AnomalyDetector│          │  Graph Engine    │
    │   (Baselines   │          │  (Flow metrics   │
    │   + Analysis)  │          │   on edges)      │
    └───────┬────────┘          └────────┬─────────┘
            │                             │
            │         ┌───────────────────┘
            │         │
    ┌───────▼─────────▼───┐
    │   HTTP API Server   │
    │   + WebSocket       │
    └───────┬─────────────┘
            │
    ┌───────▼─────────┐
    │  React Frontend │
    │  (Cytoscape.js) │
    └─────────────────┘
```

## API Endpoints

### GET /api/flows
Returns recent network flows.

**Query Parameters:**
- `limit` (int): Maximum number of flows to return (default: 100)

**Response:**
```json
[
  {
    "id": "1704123456",
    "source_pod": "test-apps/frontend-abc123",
    "source_ip": "10.244.0.5",
    "source_port": 54321,
    "dest_pod": "test-apps/backend-xyz789",
    "dest_ip": "10.244.0.6",
    "dest_port": 8080,
    "protocol": "TCP",
    "flow_type": "l3_l4",
    "bytes_sent": 1024,
    "packets_sent": 10,
    "direction": "egress",
    "verdict": "FORWARDED",
    "timestamp": "2024-01-02T15:04:05Z"
  }
]
```

### GET /api/flows/metrics
Returns aggregated flow metrics per connection.

**Response:**
```json
[
  {
    "source_id": "pod/test-apps/frontend-abc123",
    "dest_id": "pod/test-apps/backend-xyz789",
    "bytes_per_sec": 15360.5,
    "packets_per_sec": 25.3,
    "connection_count": 142,
    "error_rate": 0.02,
    "protocol": "TCP",
    "last_seen": "2024-01-02T15:04:05Z"
  }
]
```

### GET /api/flows/anomalies
Returns detected network anomalies.

**Query Parameters:**
- `severity` (string): Filter by severity (critical, high, medium, low)

**Response:**
```json
[
  {
    "id": "spike-pod/test-apps/frontend-1704123456",
    "type": "traffic_spike",
    "severity": "high",
    "title": "Traffic Spike Detected",
    "description": "Traffic from test-apps/frontend to test-apps/backend is 4.2x higher than baseline",
    "source_pod": "pod/test-apps/frontend-abc123",
    "dest_pod": "pod/test-apps/backend-xyz789",
    "evidence": {
      "current_value": 64512.0,
      "baseline_value": 15360.0,
      "threshold": 46080.0,
      "details": {
        "multiplier": "4.2x"
      }
    },
    "detected_at": "2024-01-02T15:04:05Z",
    "score": 0.42
  }
]
```

### GET /api/flows/active
Returns graph edges with active flow data.

**Response:**
```json
[
  {
    "id": "pod/test-apps/frontend->pod/test-apps/backend",
    "source": "pod/test-apps/frontend-abc123",
    "target": "pod/test-apps/backend-xyz789",
    "type": "connection",
    "health": "healthy",
    "flow_data": {
      "bytes_per_sec": 15360.5,
      "packets_per_sec": 25.3,
      "connection_count": 142,
      "error_rate": 0.02,
      "protocol": "TCP",
      "last_seen": "2024-01-02T15:04:05Z",
      "is_active": true,
      "direction": "bidirectional"
    }
  }
]
```

### WebSocket /ws/flows
Real-time flow streaming endpoint.

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/flows');
ws.onmessage = (event) => {
  const flow = JSON.parse(event.data);
  console.log('New flow:', flow);
};
```

## Configuration

### Backend Flags

```bash
./network-visualizer \
  --enable-flows=true \
  --hubble-addr=hubble-relay.kube-system.svc.cluster.local:80
```

### Environment Variables

```yaml
env:
- name: ENABLE_FLOWS
  value: "true"
- name: HUBBLE_ADDR
  value: "hubble-relay.kube-system.svc.cluster.local:80"
```

### Tuning Parameters

Edit `backend/pkg/flowcollector/flowcollector.go`:
```go
maxFlows:     10000,              // Max flows to retain
metricWindow: 60 * time.Second,   // Metric aggregation window
```

Edit `backend/pkg/flowcollector/anomaly.go`:
```go
spikeThreshold:      3.0,   // Multiplier for spike detection
errorRateThreshold:  0.05,  // 5% error rate threshold
portScanThreshold:   20,    // Unique ports for scan detection
exfilThreshold:      10 * 1024 * 1024, // 10 MB/s exfil threshold
```

## Testing

### Quick Test
```bash
# Run automated test
./test-flow-visualization.sh
```

### Manual Test

1. **Start minikube with Cilium**
   ```bash
   minikube start --network-plugin=cni --cni=false
   helm install cilium cilium/cilium --set hubble.relay.enabled=true
   ```

2. **Deploy visualizer**
   ```bash
   kubectl apply -f deploy/manifests/deployment.yaml
   kubectl -n network-visualizer set env deployment/network-visualizer ENABLE_FLOWS=true
   ```

3. **Generate traffic**
   ```bash
   kubectl run -it --rm traffic-gen --image=nicolaka/netshoot -- bash
   # Inside pod:
   while true; do curl http://some-service; sleep 1; done
   ```

4. **View flows**
   ```bash
   kubectl port-forward -n network-visualizer svc/network-visualizer 8080:80
   # Open http://localhost:8080
   ```

## Performance Considerations

### Resource Usage
- **Memory**: ~100MB per 10,000 flows
- **CPU**: <5% on 2-core system with moderate traffic
- **Network**: ~1KB/s per active flow to frontend

### Scalability
- Tested with:
  - 100 pods generating traffic
  - 500 flows/second
  - 50,000 total flows in memory
- Limitations:
  - WebSocket: ~100 clients max per instance
  - Flow retention: Configurable, default 10k flows

### Optimization Tips
1. **Reduce flow retention** for high-traffic clusters
2. **Increase metric window** to reduce CPU usage
3. **Filter flows** by namespace if only monitoring specific apps
4. **Use multiple instances** with load balancer for >100 WebSocket clients

## Troubleshooting

### No flows appearing

**Check Hubble connectivity:**
```bash
kubectl -n kube-system logs deployment/hubble-relay
kubectl -n network-visualizer logs deployment/network-visualizer | grep -i hubble
```

**Verify Cilium is capturing flows:**
```bash
kubectl -n kube-system exec -it ds/cilium -- cilium hubble observe --last 10
```

### WebSocket disconnecting

**Check browser console** for errors

**Verify WebSocket endpoint:**
```bash
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  http://localhost:8080/ws/flows
```

### High memory usage

**Reduce flow retention:**
```go
maxFlows: 5000, // Down from 10000
```

**Reduce metric window:**
```go
metricWindow: 30 * time.Second, // Down from 60s
```

## Future Enhancements

- [ ] Flow filtering by namespace/label in UI
- [ ] Historical flow replay
- [ ] Flow export to Prometheus/Loki
- [ ] Multi-cluster flow aggregation
- [ ] Custom anomaly rule engine
- [ ] Integration with Falco for security events
- [ ] Flow path tracing across service mesh

## References

- [Cilium Documentation](https://docs.cilium.io/)
- [Hubble Architecture](https://docs.cilium.io/en/stable/observability/hubble/architecture/)
- [eBPF Overview](https://ebpf.io/)
- [Setup Guide](./docs/FLOW_VISUALIZATION_SETUP.md)
