#!/bin/bash

# Demo script for Kubernetes Network Visualizer
# This script demonstrates the tool's functionality

set -e

echo "🚀 Kubernetes Network Visualizer Demo"
echo "======================================"

# Build the application
echo "📦 Building the application..."
make build

# Check if the binary was created
if [ ! -f "bin/k8-network-visualizer" ]; then
    echo "❌ Build failed - binary not found"
    exit 1
fi

echo "✅ Build successful!"

# Show help
echo ""
echo "📖 Showing help information:"
echo "----------------------------"
./bin/k8-network-visualizer -h

echo ""
echo "🧪 Running tests..."
echo "-------------------"
go test ./...

echo ""
echo "✅ Demo completed successfully!"
echo ""
echo "💡 Next steps:"
echo "   1. Connect to a Kubernetes cluster:"
echo "      export KUBECONFIG=/path/to/your/kubeconfig"
echo ""
echo "   2. Try the CLI interface:"
echo "      ./bin/k8-network-visualizer"
echo ""
echo "   3. Start the web interface:"
echo "      ./bin/k8-network-visualizer -output=web"
echo "      Then open http://localhost:8080"
echo ""
echo "   4. Export to JSON:"
echo "      ./bin/k8-network-visualizer -output=json > topology.json"
echo ""
echo "🔧 For development:"
echo "   make dev    # Run full development workflow"
echo "   make run    # Build and run CLI version" 
echo "   make run-web # Build and run web version"