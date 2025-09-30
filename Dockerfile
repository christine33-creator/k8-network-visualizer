# Multi-stage build
# Stage 1: Build the Go backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy backend code
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

# Build the backend binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o network-visualizer cmd/main.go

# Stage 2: Build the React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Copy frontend code
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ .

# Build the frontend
RUN npm run build

# Stage 3: Build the CLI tool
FROM golang:1.25-alpine AS cli-builder

WORKDIR /app

# Copy CLI code
COPY cli/go.mod cli/go.sum ./
RUN go mod download

COPY cli/ .

# Build the CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-netvis cmd/main.go

# Stage 4: Final image
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/network-visualizer .

# Copy frontend build
COPY --from=frontend-builder /app/dist ./frontend/build

# Copy CLI binary
COPY --from=cli-builder /app/k8s-netvis /usr/local/bin/

# Create a non-root user
RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose the port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run the backend server
CMD ["./network-visualizer"]
