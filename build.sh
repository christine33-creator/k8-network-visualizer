#!/bin/bash

# Build script for Kubernetes Network Visualizer

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Kubernetes Network Visualizer Build Script ===${NC}"

# Check prerequisites
echo -e "\n${YELLOW}Checking prerequisites...${NC}"

if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Please install Go 1.21+${NC}"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo -e "${RED}npm is not installed. Please install Node.js 18+${NC}"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed. Please install Docker${NC}"
    exit 1
fi

echo -e "${GREEN}Prerequisites check passed!${NC}"

# Build backend
echo -e "\n${YELLOW}Building backend...${NC}"
cd backend
go mod download
go build -o ../bin/network-visualizer cmd/main.go
echo -e "${GREEN}Backend built successfully!${NC}"

# Build frontend
echo -e "\n${YELLOW}Building frontend...${NC}"
cd ../frontend
npm ci
npm run build
echo -e "${GREEN}Frontend built successfully!${NC}"

# Build CLI
echo -e "\n${YELLOW}Building CLI...${NC}"
cd ../cli
go mod download
go build -o ../bin/k8s-netvis cmd/main.go
echo -e "${GREEN}CLI built successfully!${NC}"

# Build Docker image
echo -e "\n${YELLOW}Building Docker image...${NC}"
cd ..
docker build -t k8s-network-visualizer:latest .
echo -e "${GREEN}Docker image built successfully!${NC}"

# Tag for registry
if [ "$1" == "--push" ]; then
    echo -e "\n${YELLOW}Pushing to registry...${NC}"
    REGISTRY=${DOCKER_REGISTRY:-"christine33creator"}
    docker tag k8s-network-visualizer:latest $REGISTRY/k8s-network-visualizer:latest
    docker push $REGISTRY/k8s-network-visualizer:latest
    echo -e "${GREEN}Image pushed to registry!${NC}"
fi

echo -e "\n${GREEN}=== Build Complete ===${NC}"
echo -e "Binaries available in: ./bin/"
echo -e "Docker image: k8s-network-visualizer:latest"
echo -e "\nTo deploy to Kubernetes:"
echo -e "  kubectl apply -f deploy/manifests/deployment.yaml"
echo -e "\nTo run locally:"
echo -e "  ./bin/network-visualizer"
echo -e "\nTo use CLI:"
echo -e "  ./bin/k8s-netvis --help"