# Universal Network Flow Visualization

## Overview

The Kubernetes Network Visualizer now uses a **CNI-agnostic and Service Mesh-agnostic** architecture for network flow visualization. This means it works on ANY Kubernetes cluster regardless of:

- ✅ CNI (Container Network Interface): Cilium, Calico, Flannel, Weave, or none
- ✅ Service Mesh: Istio, Linkerd, Consul, or none  
- ✅ Cloud Provider: AWS, Azure, GCP, on-prem, or local (minikube/kind)

## How It Works

### Universal Flow Collection (Default)

The system uses **kernel-level observability** that exists on every Linux system:

1. **Connection Tracking (conntrack)**
   - Linux kernel tracks ALL network connections
   - Available via `/proc/net/nf_conntrack` or `conntrack` command
   - Provides: Source/Dest IPs, ports, protocols, byte/packet counts, connection state
   - Works with ANY networking setup

2. **iptables Statistics**
   - Every Linux system uses iptables for packet filtering
   - Provides: Packet counts, byte counts, DROP statistics
   - Command: `iptables -L -n -v -x`
   - Supplements flow data with firewall statistics

3. **Pod IP Resolution**
   - Maps IP addresses to Kubernetes pods using K8s API
   - Cached in memory for performance
   - Updated every 10 seconds

### Data Collection Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                    │
│                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │    Pod A    │  │    Pod B    │  │    Pod C    │    │
│  │  10.244.0.5 │  │  10.244.0.6 │  │  10.244.0.7 │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
│         │                │                │             │
│         └────────────────┴────────────────┘             │
│                         │                                │
│  ┌──────────────────────▼──────────────────────────┐   │
│  │          Linux Kernel (conntrack/iptables)       │   │
│  │                                                   │   │
│  │  /proc/net/nf_conntrack:                         │   │
│  │  tcp 6 ESTAB src=10.244.0.5 dst=10.244.0.6 ...  │   │
│  │  tcp 6 ESTAB src=10.244.0.6 dst=10.244.0.7 ...  │   │
│  └──────────────────────▲──────────────────────────┘   │
│                         │                                │
│  ┌──────────────────────┴──────────────────────────┐   │
│  │      Universal Flow Collector (hostNetwork)      │   │
│  │      - Reads /proc/net/nf_conntrack              │   │
│  │      - Parses iptables statistics                │   │
│  │      - Resolves IPs to pod names via K8s API     │   │
│  │      - Aggregates flows                          │   │
│  └──────────────────────┬──────────────────────────┘   │
│                         │                                │
│  ┌──────────────────────▼──────────────────────────┐   │
│  │          Network Visualizer Backend               │   │
│  │      - Flow metrics API                          │   │
│  │      - Anomaly detection                         │   │
│  │      - Graph visualization                       │   │
│  └───────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Enhanced Collectors (Auto-Detected)

If available, the system automatically uses enhanced collectors:

1. **Cilium Hubble** (if Cilium CNI detected)
   - eBPF-based L7 visibility
   - HTTP/gRPC protocol details
   - DNS request tracking
   - Enhanced drop reasons

2. **Istio/Envoy Metrics** (if Istio detected)
   - Service mesh telemetry
   - mTLS connection info
   - Request/response metrics
   - Retry and timeout data

3. **Calico Felix** (if Calico detected)
   - CNI-native flow logs
   - Network policy enforcement data
   - BGP routing information

**Priority**: Cilium > Istio > Calico > Universal (always works)

## Deployment Requirements

### Permissions Required

The universal collector needs access to kernel networking data:

```yaml
securityContext:
  capabilities:
    add:
    - NET_ADMIN  # Read conntrack/iptables
    - NET_RAW    # Packet inspection
hostNetwork: true  # Access host network stack
```

### Why hostNetwork?

- The `conntrack` table is per-node, not per-pod
- Each node tracks connections for all pods on that node  
- To see all cluster traffic, we need access to each node's network namespace

### DaemonSet vs Deployment

**Current**: Single deployment with `hostNetwork: true`
- Works for single-node clusters (minikube, kind)
- Collects flows from the node it's running on

**Production**: Convert to DaemonSet for multi-node clusters
- One collector pod per node
- Each collects local node's conntrack data
- Aggregates at backend for cluster-wide view

## What You Get

### Flow Data (Universal Mode)

```json
{
  "source_pod": "default/nginx-abc123",
  "source_ip": "10.244.0.5",
  "source_port": 45678,
  "dest_pod": "default/backend-xyz789",
  "dest_ip": "10.244.0.6",
  "dest_port": 8080,
  "protocol": "TCP",
  "bytes_sent": 1048576,
  "packets_sent": 724,
  "bytes_per_sec": 52428.8,
  "packets_per_sec": 36.2,
  "direction": "egress",
  "verdict": "ACCEPT",
  "timestamp": "2025-11-13T19:30:00Z"
}
```

### Aggregated Metrics

```json
{
  "default/nginx→default/backend": {
    "bytes_per_sec": 52428.8,
    "packets_per_sec": 36.2,
    "connection_count": 12,
    "error_rate": 0.0,
    "protocol": "TCP",
    "is_active": true,
    "direction": "bidirectional"
  }
}
```

### Anomaly Detection

Still works! The anomaly detector analyzes:
- Traffic spikes/drops
- Port scanning
- Data exfiltration patterns
- Unusual protocols
- High error rates
- Lateral movement
- DNS tunneling

## Limitations

### Universal Mode (conntrack/iptables)

**✅ What Works:**
- Source/destination IPs and ports
- Protocol (TCP/UDP/ICMP)
- Byte and packet counts
- Connection state (ESTABLISHED, TIME_WAIT, etc.)
- Connection tracking
- Pod-to-pod flow visualization

**❌ What's Limited:**
- No L7 protocol visibility (HTTP, gRPC, etc.)
- No request/response details
- No DNS query content (only connections)
- No mTLS information
- Drop reasons less detailed

### Enhanced Mode (Cilium/Istio/Calico)

All limitations above are removed with:
- Cilium Hubble: Full L7 visibility via eBPF
- Istio: Service mesh telemetry
- Calico: CNI-native flow logs

## Testing

### Test on Bare Minikube

```bash
# Start minikube without special CNI
minikube start --cni=auto

# Deploy visualizer
kubectl apply -f deploy/manifests/deployment.yaml

# Check flow collection
kubectl -n network-visualizer logs deployment/network-visualizer | grep "Universal"
# Should see: "✓ Universal flow collection using kernel conntrack"

# Port-forward
kubectl -n network-visualizer port-forward svc/network-visualizer 8080:80

# Test API
curl http://localhost:8080/api/flows
curl http://localhost:8080/api/flows/metrics
```

### Generate Test Traffic

```bash
# Deploy test apps
kubectl apply -f deploy/test-apps.yaml

# Watch flows appear
watch -n 1 'curl -s http://localhost:8080/api/flows | jq ".[] | {source_pod, dest_pod, protocol, bytes_per_sec}"'
```

## Comparison with CNI-Specific Approaches

| Feature | Universal (conntrack) | Cilium Hubble | Istio | Calico |
|---------|----------------------|---------------|-------|--------|
| **Works with any CNI** | ✅ | ❌ (Cilium only) | ✅ | ❌ (Calico only) |
| **No service mesh needed** | ✅ | ✅ | ❌ | ✅ |
| **L3/L4 visibility** | ✅ | ✅ | ✅ | ✅ |
| **L7 visibility** | ❌ | ✅ | ✅ | Limited |
| **HTTP/gRPC details** | ❌ | ✅ | ✅ | ❌ |
| **DNS tracking** | Connections only | ✅ Full queries | ✅ | ✅ |
| **Drop reasons** | Limited | ✅ Detailed | ✅ | ✅ |
| **mTLS info** | ❌ | ❌ | ✅ | ❌ |
| **Performance overhead** | Very low | Low (eBPF) | Medium (sidecars) | Low |
| **Setup complexity** | Minimal | Requires Cilium | Requires Istio | Requires Calico |

## Migration from Cilium-Only Version

### Old Way (Cilium Required)

```yaml
env:
- name: ENABLE_FLOWS
  value: "true"
- name: HUBBLE_ADDR
  value: "hubble-relay.kube-system.svc.cluster.local:80"
```

This would fail if Cilium wasn't installed.

### New Way (Universal)

```yaml
env:
- name: ENABLE_FLOWS
  value: "true"
# No HUBBLE_ADDR needed - auto-detects best method

securityContext:
  capabilities:
    add: [NET_ADMIN, NET_RAW]
hostNetwork: true
```

Works everywhere! Auto-uses Cilium if available.

## Future Enhancements

1. **DaemonSet Mode**: Collect from all nodes in multi-node clusters
2. **eBPF Standalone**: Custom eBPF programs without Cilium dependency
3. **Packet Capture**: Optional tcpdump integration for deep analysis
4. **Flow Export**: Export to external systems (Kafka, S3, etc.)
5. **Historical Analysis**: Store flows for trend analysis

## Summary

**Before**: Required Cilium CNI with Hubble enabled  
**Now**: Works with ANY setup, auto-enhances if Cilium/Istio/Calico available

This makes the tool universally applicable while still taking advantage of advanced CNI/service mesh features when present.
