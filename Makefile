# Kubernetes Network Visualizer Makefile

.PHONY: build clean test lint fmt vet run help

# Build variables
BINARY_NAME=k8-network-visualizer
BUILD_DIR=bin
SOURCE_DIR=cmd
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
all: build

## Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(SOURCE_DIR)/main.go

## Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	go clean

## Run tests
test:
	@echo "Running tests..."
	go test -v ./...

## Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

## Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

## Run the application (CLI mode)
run: build
	@echo "Running $(BINARY_NAME) in CLI mode..."
	./$(BUILD_DIR)/$(BINARY_NAME)

## Run the application (Web mode)
run-web: build
	@echo "Running $(BINARY_NAME) in web mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) -output=web

## Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download

## Development workflow
dev: fmt vet tidy test build

## Show help
help:
	@echo "Available targets:"
	@echo "  build     - Build the application"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  lint      - Run linter (requires golangci-lint)"
	@echo "  fmt       - Format code"
	@echo "  vet       - Run go vet"
	@echo "  tidy      - Tidy dependencies"
	@echo "  run       - Build and run CLI version"
	@echo "  run-web   - Build and run web version"
	@echo "  deps      - Install dependencies"
	@echo "  dev       - Run development workflow (fmt, vet, tidy, test, build)"
	@echo "  help      - Show this help"