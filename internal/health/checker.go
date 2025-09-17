package health

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/christine33-creator/k8-network-visualizer/pkg/models"
)

// Checker performs network connectivity health checks
type Checker struct{}

// NewChecker creates a new health checker
func NewChecker() *Checker {
	return &Checker{}
}

// CheckConnectivity performs connectivity checks between network components
func (c *Checker) CheckConnectivity(ctx context.Context, topology *models.NetworkTopology) ([]models.HealthCheck, error) {
	var healthChecks []models.HealthCheck

	// Check service to endpoint connectivity
	for _, service := range topology.Services {
		for _, endpoint := range service.Endpoints {
			for _, port := range service.Ports {
				check := c.checkTCPConnection(ctx, endpoint, port.Port, port.Protocol)
				check.Source = fmt.Sprintf("%s/%s", service.Namespace, service.Name)
				check.Destination = endpoint
				healthChecks = append(healthChecks, check)
			}
		}
	}

	// Check pod to pod connectivity
	for i, pod1 := range topology.Pods {
		for j, pod2 := range topology.Pods {
			if i >= j || pod1.Namespace != pod2.Namespace {
				continue // Skip same pod and cross-namespace for now
			}

			for _, port := range pod2.Ports {
				check := c.checkTCPConnection(ctx, pod2.IP, port.Port, port.Protocol)
				check.Source = fmt.Sprintf("%s/%s", pod1.Namespace, pod1.Name)
				check.Destination = fmt.Sprintf("%s/%s", pod2.Namespace, pod2.Name)
				healthChecks = append(healthChecks, check)
			}
		}
	}

	return healthChecks, nil
}

func (c *Checker) checkTCPConnection(ctx context.Context, host string, port int32, protocol string) models.HealthCheck {
	check := models.HealthCheck{
		Port:      port,
		Protocol:  protocol,
		Timestamp: time.Now(),
	}

	if protocol != "TCP" {
		check.Status = "skipped"
		check.Error = "Only TCP connectivity checks supported"
		return check
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	check.Latency = time.Since(start)

	if err != nil {
		check.Status = "failed"
		check.Error = err.Error()
	} else {
		check.Status = "success"
		conn.Close()
	}

	return check
}

// CheckDNSResolution checks if DNS resolution is working for services
func (c *Checker) CheckDNSResolution(ctx context.Context, topology *models.NetworkTopology) ([]models.HealthCheck, error) {
	var healthChecks []models.HealthCheck

	for _, service := range topology.Services {
		serviceName := fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace)

		start := time.Now()
		ips, err := net.LookupIP(serviceName)
		latency := time.Since(start)

		check := models.HealthCheck{
			Source:      "dns-resolver",
			Destination: serviceName,
			Protocol:    "DNS",
			Latency:     latency,
			Timestamp:   time.Now(),
		}

		if err != nil {
			check.Status = "failed"
			check.Error = err.Error()
		} else if len(ips) == 0 {
			check.Status = "failed"
			check.Error = "No IPs resolved"
		} else {
			check.Status = "success"
		}

		healthChecks = append(healthChecks, check)
	}

	return healthChecks, nil
}
