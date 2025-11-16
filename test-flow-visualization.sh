#!/bin/bash

# Network Traffic Flow Visualization - Automated Test Script
# This script sets up minikube with Cilium, deploys the visualizer, and runs tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    echo_info "Checking prerequisites..."
    
    command -v minikube >/dev/null 2>&1 || { echo_error "minikube is required but not installed."; exit 1; }
    command -v kubectl >/dev/null 2>&1 || { echo_error "kubectl is required but not installed."; exit 1; }
    command -v helm >/dev/null 2>&1 || { echo_error "helm is required but not installed."; exit 1; }
    command -v docker >/dev/null 2>&1 || { echo_error "docker is required but not installed."; exit 1; }
    
    echo_info "All prerequisites met"
}

# Setup minikube with Cilium
setup_minikube() {
    echo_info "Setting up minikube with Cilium..."
    
    # Check if minikube is already running
    if minikube status >/dev/null 2>&1; then
        echo_warn "Minikube is already running. Delete it? (y/n)"
        read -r response
        if [[ "$response" == "y" ]]; then
            minikube delete
        else
            echo_info "Using existing minikube cluster"
            return
        fi
    fi
    
    # Start minikube
    echo_info "Starting minikube..."
    minikube start \
        --network-plugin=cni \
        --cni=false \
        --memory=6144 \
        --cpus=4 \
        --driver=docker
    
    # Install Cilium
    echo_info "Installing Cilium with Hubble..."
    helm repo add cilium https://helm.cilium.io/ 2>/dev/null || true
    helm repo update
    
    helm install cilium cilium/cilium \
        --version 1.14.0 \
        --namespace kube-system \
        --set hubble.relay.enabled=true \
        --set hubble.ui.enabled=true \
        --set hubble.metrics.enabled="{dns,drop,tcp,flow,icmp,http}"
    
    # Wait for Cilium
    echo_info "Waiting for Cilium to be ready..."
    kubectl -n kube-system wait --for=condition=available --timeout=300s deployment/cilium-operator
    kubectl -n kube-system wait --for=condition=ready --timeout=300s pod -l k8s-app=hubble-relay
    
    echo_info "Cilium installed successfully"
}

# Build and deploy network visualizer
deploy_visualizer() {
    echo_info "Building and deploying network visualizer..."
    
    # Set docker env for minikube
    eval $(minikube docker-env)
    
    # Build backend
    echo_info "Building backend..."
    cd backend
    go mod download
    go mod tidy
    cd ..
    
    # Build Docker image
    echo_info "Building Docker image..."
    docker build -t k8s-network-visualizer:latest -f Dockerfile .
    
    # Build frontend (if needed)
    if [ -d "frontend" ]; then
        echo_info "Building frontend..."
        cd frontend
        if [ ! -d "node_modules" ]; then
            npm install
        fi
        npm run build
        cd ..
    fi
    
    # Deploy to Kubernetes
    echo_info "Deploying to Kubernetes..."
    kubectl apply -f deploy/manifests/deployment.yaml
    
    # Enable flow collection
    echo_info "Enabling flow collection..."
    kubectl -n network-visualizer wait --for=condition=available --timeout=180s deployment/network-visualizer
    kubectl -n network-visualizer set env deployment/network-visualizer \
        ENABLE_FLOWS=true \
        HUBBLE_ADDR=hubble-relay.kube-system.svc.cluster.local:80
    
    # Wait for rollout
    kubectl -n network-visualizer rollout status deployment/network-visualizer --timeout=180s
    
    echo_info "Network visualizer deployed successfully"
}

# Deploy test applications
deploy_test_apps() {
    echo_info "Deploying test applications..."
    
    if [ -f "deploy/test-apps.yaml" ]; then
        kubectl apply -f deploy/test-apps.yaml
        kubectl -n test-apps wait --for=condition=ready --timeout=180s pod --all
        echo_info "Test applications deployed"
    else
        echo_warn "test-apps.yaml not found, creating basic test deployment..."
        
        kubectl create namespace test-apps --dry-run=client -o yaml | kubectl apply -f -
        
        cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: test-apps
spec:
  replicas: 2
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-service
  namespace: test-apps
spec:
  selector:
    app: frontend
  ports:
  - port: 80
    targetPort: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-api
  namespace: test-apps
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend-api
  template:
    metadata:
      labels:
        app: backend-api
    spec:
      containers:
      - name: api
        image: hashicorp/http-echo
        args:
        - "-text=Hello from backend"
        ports:
        - containerPort: 5678
---
apiVersion: v1
kind: Service
metadata:
  name: backend-api-service
  namespace: test-apps
spec:
  selector:
    app: backend-api
  ports:
  - port: 8080
    targetPort: 5678
EOF
        kubectl -n test-apps wait --for=condition=ready --timeout=180s pod --all
    fi
}

# Generate test traffic
generate_traffic() {
    echo_info "Generating test traffic..."
    
    cat <<EOF | kubectl apply -f -
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
        echo "Starting traffic generation..."
        while true; do
          # HTTP traffic
          curl -s http://backend-api-service:8080/ >/dev/null 2>&1 || true
          curl -s http://frontend-service:80/ >/dev/null 2>&1 || true
          
          # DNS lookups
          nslookup backend-api-service >/dev/null 2>&1 || true
          nslookup frontend-service >/dev/null 2>&1 || true
          
          # TCP connections
          nc -zv -w 1 backend-api-service 8080 >/dev/null 2>&1 || true
          
          sleep 2
        done
EOF
    
    kubectl -n test-apps wait --for=condition=ready --timeout=60s pod/traffic-generator
    echo_info "Traffic generator started"
}

# Run tests
run_tests() {
    echo_info "Running flow visualization tests..."
    
    # Port forward the service
    echo_info "Setting up port-forward..."
    kubectl -n network-visualizer port-forward svc/network-visualizer 8080:80 >/dev/null 2>&1 &
    PF_PID=$!
    sleep 5
    
    # Test 1: Health check
    echo_info "Test 1: Health check"
    if curl -s http://localhost:8080/api/health | grep -q "healthy"; then
        echo_info "✓ Health check passed"
    else
        echo_error "✗ Health check failed"
    fi
    
    # Test 2: Get flows
    echo_info "Test 2: Get flows (waiting 30s for flows to accumulate...)"
    sleep 30
    FLOW_COUNT=$(curl -s http://localhost:8080/api/flows?limit=10 | jq '. | length')
    if [ "$FLOW_COUNT" -gt 0 ]; then
        echo_info "✓ Flows detected: $FLOW_COUNT flows"
    else
        echo_warn "✗ No flows detected yet"
    fi
    
    # Test 3: Get flow metrics
    echo_info "Test 3: Get flow metrics"
    METRIC_COUNT=$(curl -s http://localhost:8080/api/flows/metrics | jq '. | length')
    if [ "$METRIC_COUNT" -gt 0 ]; then
        echo_info "✓ Flow metrics available: $METRIC_COUNT metrics"
    else
        echo_warn "✗ No flow metrics available yet"
    fi
    
    # Test 4: Generate traffic spike and check anomalies
    echo_info "Test 4: Testing anomaly detection (generating traffic spike...)"
    kubectl -n test-apps exec traffic-generator -- bash -c '
        for i in {1..50}; do curl -s http://backend-api-service:8080/ >/dev/null 2>&1 & done
        wait
    ' >/dev/null 2>&1 || true
    
    sleep 10
    ANOMALY_COUNT=$(curl -s http://localhost:8080/api/flows/anomalies | jq '. | length')
    if [ "$ANOMALY_COUNT" -gt 0 ]; then
        echo_info "✓ Anomalies detected: $ANOMALY_COUNT anomalies"
        curl -s http://localhost:8080/api/flows/anomalies | jq -r '.[] | "  - \(.title): \(.description)"' | head -5
    else
        echo_warn "✗ No anomalies detected (may need more time for baseline)"
    fi
    
    # Test 5: Active flows
    echo_info "Test 5: Get active flows"
    ACTIVE_FLOWS=$(curl -s http://localhost:8080/api/flows/active | jq '. | length')
    echo_info "Active flows: $ACTIVE_FLOWS"
    
    # Cleanup port-forward
    kill $PF_PID 2>/dev/null || true
    
    echo_info "All tests completed!"
}

# Show access information
show_access_info() {
    echo_info "====================================="
    echo_info "Setup Complete!"
    echo_info "====================================="
    echo ""
    echo_info "To access the Network Visualizer UI:"
    echo "  kubectl -n network-visualizer port-forward svc/network-visualizer 8080:80"
    echo "  Then open: http://localhost:8080"
    echo ""
    echo_info "To view Hubble UI:"
    echo "  kubectl -n kube-system port-forward svc/hubble-ui 12000:80"
    echo "  Then open: http://localhost:12000"
    echo ""
    echo_info "Useful commands:"
    echo "  # View flows"
    echo "  kubectl -n kube-system exec -it ds/cilium -- cilium hubble observe --last 20"
    echo ""
    echo "  # View visualizer logs"
    echo "  kubectl -n network-visualizer logs -f deployment/network-visualizer"
    echo ""
    echo "  # View traffic generator"
    echo "  kubectl -n test-apps logs -f traffic-generator"
    echo ""
    echo_info "To cleanup:"
    echo "  kubectl delete -f deploy/manifests/deployment.yaml"
    echo "  kubectl delete namespace test-apps"
    echo "  minikube delete"
}

# Main execution
main() {
    echo_info "Starting Network Flow Visualization Setup"
    echo_info "=========================================="
    
    check_prerequisites
    setup_minikube
    deploy_visualizer
    deploy_test_apps
    generate_traffic
    run_tests
    show_access_info
}

# Run main function
main
