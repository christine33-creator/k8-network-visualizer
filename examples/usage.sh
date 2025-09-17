# Example usage scenarios for k8-network-visualizer

## Scenario 1: Debug AKS connectivity issues
# Analyze production namespace for connectivity problems
./k8-network-visualizer -namespace=production -verbose

## Scenario 2: Export topology for analysis
# Generate JSON export for further processing
./k8-network-visualizer -output=json > aks-topology.json

## Scenario 3: Web visualization for team review
# Start web interface for collaborative debugging
./k8-network-visualizer -output=web -port=8080

## Scenario 4: Analyze specific cluster
# Use custom kubeconfig for different clusters
./k8-network-visualizer -kubeconfig=/path/to/aks-cluster.config

## Scenario 5: Monitor system namespaces
# Check kube-system for infrastructure issues
./k8-network-visualizer -namespace=kube-system

## Scenario 6: Cross-namespace analysis
# Analyze all namespaces for global view
./k8-network-visualizer

## Common AKS debugging workflows:

### 1. Service mesh connectivity
# Check if your service mesh is properly configured
./k8-network-visualizer -namespace=istio-system

### 2. CNI plugin issues
# Analyze node and pod networking
./k8-network-visualizer -verbose | grep -E "(Node|CIDR)"

### 3. Network policy validation
# Export policies for review
./k8-network-visualizer -output=json | jq '.policies'

### 4. LoadBalancer service troubleshooting
# Check service configurations
./k8-network-visualizer -output=json | jq '.services[] | select(.type=="LoadBalancer")'