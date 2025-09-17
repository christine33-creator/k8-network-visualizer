# Troubleshooting Guide

## Common Issues and Solutions

### 1. "failed to create kubernetes client" Error

**Problem**: Cannot connect to Kubernetes cluster

**Solutions**:
- Verify `kubectl` is working: `kubectl cluster-info`
- Check kubeconfig path: `echo $KUBECONFIG`
- Ensure proper RBAC permissions for your user/service account
- For AKS: `az aks get-credentials --resource-group myResourceGroup --name myAKSCluster`

### 2. "failed to get network topology" Error

**Problem**: Cannot retrieve cluster resources

**Solutions**:
- Check if you have read permissions for the namespace
- Verify the namespace exists: `kubectl get namespaces`
- For specific namespace: `kubectl auth can-i get pods --namespace=target-namespace`

### 3. Web Interface Not Loading

**Problem**: Cannot access web interface on localhost:8080

**Solutions**:
- Check if port is available: `netstat -an | grep 8080`
- Try different port: `./k8-network-visualizer -output=web -port=8081`
- Check firewall settings
- Verify application started successfully (check console output)

### 4. Empty or Incomplete Topology

**Problem**: Missing nodes, pods, or services in output

**Solutions**:
- Verify resources exist: `kubectl get nodes,pods,services --all-namespaces`
- Check RBAC permissions for cluster-level resources
- Try without namespace filter: `./k8-network-visualizer` (all namespaces)
- Enable verbose logging: `./k8-network-visualizer -verbose`

### 5. Network Policy Issues

**Problem**: Network policies not showing up or incorrect

**Solutions**:
- Verify network policies exist: `kubectl get networkpolicies --all-namespaces`
- Check if CNI supports network policies (Calico, Cilium, etc.)
- Ensure proper RBAC for networking.k8s.io/v1 resources

### 6. Health Check Failures

**Problem**: All health checks show as failed

**Solutions**:
- Verify pods are running: `kubectl get pods --all-namespaces`
- Check service endpoints: `kubectl get endpoints`
- Network connectivity issues between cluster nodes
- DNS resolution problems within cluster

## AKS-Specific Troubleshooting

### 1. Azure CNI Issues

**Problem**: Overlapping CIDR or IP exhaustion

**Diagnosis**:
```bash
# Check node CIDR allocations
./k8-network-visualizer -output=json | jq '.nodes[].cidrs'

# Check pod IP ranges
kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}'
```

**Solutions**:
- Review AKS subnet size
- Check Azure Virtual Network configuration
- Consider using kubenet instead of Azure CNI for larger clusters

### 2. Load Balancer Issues

**Problem**: External services not reachable

**Diagnosis**:
```bash
# Check LoadBalancer services
./k8-network-visualizer -output=json | jq '.services[] | select(.type=="LoadBalancer")'

# Verify Azure Load Balancer configuration
kubectl get services --all-namespaces -o wide
```

### 3. Network Security Group (NSG) Blocks

**Problem**: Traffic blocked by Azure NSG rules

**Diagnosis**:
- Check NSG rules in Azure portal
- Verify subnet associations
- Review AKS-generated security group rules

### 4. Konnectivity Issues

**Problem**: Node-to-API server connectivity problems

**Diagnosis**:
```bash
# Check konnectivity agent pods
kubectl get pods -n kube-system | grep konnectivity

# Analyze system namespace
./k8-network-visualizer -namespace=kube-system
```

## Performance Issues

### 1. Slow Topology Discovery

**Problem**: Application takes long time to discover topology

**Solutions**:
- Limit namespace scope: `-namespace=specific-namespace`
- Large clusters may need timeout adjustments
- Consider running during off-peak hours

### 2. Web Interface Performance

**Problem**: Web visualization is slow or unresponsive

**Solutions**:
- Reduce the number of visualized components
- Use JSON output for large topologies
- Consider limiting to specific namespaces

## Getting Help

If you continue to experience issues:

1. Enable verbose logging: `-verbose`
2. Capture the full error output
3. Verify your Kubernetes cluster is healthy
4. Check the [project issues](https://github.com/christine33-creator/k8-network-visualizer/issues)
5. Provide cluster information when reporting bugs:
   - Kubernetes version: `kubectl version`
   - CNI plugin in use
   - Cloud provider (AKS, EKS, GKE, etc.)
   - Cluster size and configuration