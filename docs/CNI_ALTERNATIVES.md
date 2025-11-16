# CNI Alternatives for Network Flow Visualization

## Current Implementation (Cilium-Specific)

The current flow visualization uses **Cilium Hubble**, which provides eBPF-based flow observability. This is the most performant and feature-rich option but requires Cilium as your CNI.

## Alternatives for Other CNIs

If you're not using Cilium, here are alternatives to implement network flow visualization:

### Option 1: Service Mesh (Recommended for Production)

**Using Istio/Linkerd:**
- **Pros**: CNI-agnostic, production-ready, rich metrics
- **Cons**: More overhead, requires sidecar proxies
- **Implementation**: 
  - Use Prometheus metrics from Envoy/Linkerd proxies
  - Query service mesh telemetry APIs
  - Similar feature set to Cilium Hubble

```go
// Example: Query Istio metrics for flow data
import (
    "github.com/prometheus/client_golang/api"
    promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func collectIstioFlows() {
    // Query: istio_tcp_sent_bytes_total, istio_tcp_received_bytes_total
    // Group by source_workload, destination_workload
}
```

### Option 2: CNI Network Plugin Metrics

**Using Calico/Weave/Flannel metrics:**
- **Pros**: Lightweight, CNI-native
- **Cons**: Limited flow detail compared to eBPF
- **Implementation**:
  - Calico: Felix Prometheus metrics (connection tracking)
  - Weave: weaveworks/scope for flow data
  - Flannel: iptables stats parsing

### Option 3: Kubernetes Network Policies + iptables Logging

**Using iptables packet counting:**
- **Pros**: Works with any CNI, no extra components
- **Cons**: Limited granularity, performance impact at scale
- **Implementation**:

```go
// Read iptables packet counters
func collectIptablesFlows() {
    cmd := exec.Command("iptables", "-L", "-nvx")
    // Parse packet/byte counters per rule
    // Match rules to pod network policies
}
```

### Option 4: Kernel Conntrack Table

**Using conntrack for connection tracking:**
- **Pros**: Works on any CNI, kernel-native
- **Cons**: No protocol-level visibility, basic metrics
- **Implementation**:

```go
import "github.com/ti-mo/conntrack"

func collectConntrackFlows() {
    conn, _ := conntrack.Dial(nil)
    flows, _ := conn.Dump(&conntrack.Filter{})
    
    for _, flow := range flows {
        // flow.TupleOrig.Proto, .IP.SourceAddress, .IP.DestinationAddress
        // flow.Counters.Packets, .Bytes
    }
}
```

### Option 5: Custom DaemonSet with tcpdump/BPF

**Using AF_PACKET sockets or classic BPF:**
- **Pros**: Full packet visibility, CNI-agnostic
- **Cons**: High CPU usage, requires privileged pods
- **Implementation**:

```go
import "github.com/google/gopacket/pcap"

func collectPacketFlows() {
    handle, _ := pcap.OpenLive("eth0", 1600, true, pcap.BlockForever)
    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
    
    for packet := range packetSource.Packets() {
        // Extract IPs, ports, protocols, bytes
        // Aggregate by pod IP pairs
    }
}
```

### Option 6: eBPF Without Cilium (Advanced)

**Using standalone eBPF programs:**
- **Pros**: Performance similar to Cilium, CNI-agnostic
- **Cons**: Complex to implement and maintain
- **Implementation**:
  - Write custom eBPF programs (C code)
  - Attach to network hooks (kprobe/tracepoint)
  - Use libraries like cilium/ebpf or iovisor/gobpf

```go
import "github.com/cilium/ebpf"

// Load custom eBPF program for flow tracking
func loadEBPF() {
    spec, _ := ebpf.LoadCollectionSpec("flow_tracker.o")
    coll, _ := ebpf.NewCollection(spec)
    // Attach to tc ingress/egress or XDP
}
```

## Recommendation by Use Case

| Use Case | Best Option | Why |
|----------|-------------|-----|
| **Production with Service Mesh** | Istio/Linkerd metrics | Already have telemetry data |
| **Production without Service Mesh** | Calico metrics or Conntrack | Lightweight, reliable |
| **Development/Testing** | iptables logging | Simple, no extra components |
| **High-Performance Requirements** | Cilium Hubble (switch CNI) | Best-in-class eBPF observability |
| **Multi-CNI Support** | Pluggable architecture (see below) | Support multiple backends |

## Making Flow Collection Pluggable

To support multiple CNIs, refactor the flow collector with an interface:

```go
// backend/pkg/flowcollector/interface.go
package flowcollector

type FlowCollectorInterface interface {
    Connect() error
    GetFlows(ctx context.Context) ([]Flow, error)
    Close() error
}

// Implementations:
// - HubbleCollector (current - Cilium)
// - IstioCollector (Envoy metrics)
// - CalicoCollector (Felix metrics)
// - ConntrackCollector (kernel conntrack)
// - GenericCollector (iptables/tcpdump fallback)
```

Then in main.go:
```go
var collector flowcollector.FlowCollectorInterface

switch os.Getenv("FLOW_COLLECTOR_TYPE") {
case "hubble":
    collector = flowcollector.NewHubbleCollector(hubbleAddr)
case "istio":
    collector = flowcollector.NewIstioCollector(prometheusAddr)
case "conntrack":
    collector = flowcollector.NewConntrackCollector()
default:
    log.Warn("No flow collector configured, flow visualization disabled")
}
```

## Migration Guide

### From Cilium to Another CNI (Keep Topology Visualization)

1. **Disable flow collection**:
   ```yaml
   # deploy/manifests/deployment.yaml
   env:
     - name: ENABLE_FLOWS
       value: "false"  # Disable Hubble flows
   ```

2. **Deploy without Cilium**:
   - Topology, health checks, probing, simulations still work
   - Flow visualization features will be unavailable

3. **Implement alternative collector** (optional):
   - Choose option from above (Istio, conntrack, etc.)
   - Add new collector implementation
   - Enable with `FLOW_COLLECTOR_TYPE=conntrack`

### Adding Flow Support to Non-Cilium Cluster

1. **Choose collection method** (see table above)
2. **Implement collector interface** in `backend/pkg/flowcollector/`
3. **Update deployment** with appropriate permissions:
   ```yaml
   # For conntrack/iptables collectors
   securityContext:
     capabilities:
       add: ["NET_ADMIN", "NET_RAW"]
   ```

## Example: Conntrack Collector Implementation

Want me to implement a basic conntrack-based flow collector that works with any CNI? It won't have the protocol visibility of Cilium but will show basic connection flows.

