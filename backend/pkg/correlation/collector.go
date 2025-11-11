package correlation

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// MetricsCollector collects metrics from various sources
type MetricsCollector struct {
	prometheusClient v1.API
	correlationEngine *CorrelationEngine
	
	// Collection intervals
	networkMetricsInterval time.Duration
	appMetricsInterval     time.Duration
	infraMetricsInterval   time.Duration
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(prometheusURL string, engine *CorrelationEngine) (*MetricsCollector, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Prometheus client: %w", err)
	}

	return &MetricsCollector{
		prometheusClient:       v1.NewAPI(client),
		correlationEngine:      engine,
		networkMetricsInterval: 30 * time.Second,
		appMetricsInterval:     30 * time.Second,
		infraMetricsInterval:   60 * time.Second,
	}, nil
}

// Start begins collecting metrics
func (mc *MetricsCollector) Start(ctx context.Context) {
	// Start network metrics collection
	go mc.collectNetworkMetrics(ctx)
	
	// Start application metrics collection
	go mc.collectApplicationMetrics(ctx)
	
	// Start infrastructure metrics collection
	go mc.collectInfrastructureMetrics(ctx)
	
	// Start correlation analysis
	go mc.runCorrelationAnalysis(ctx)
}

// collectNetworkMetrics collects network-related metrics
func (mc *MetricsCollector) collectNetworkMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.networkMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "network_packets_dropped", 
				`rate(node_network_receive_drop_total[5m])`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "network_errors", 
				`rate(node_network_receive_errs_total[5m])`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "network_bytes_sent", 
				`rate(node_network_transmit_bytes_total[5m])`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "network_bytes_received", 
				`rate(node_network_receive_bytes_total[5m])`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "tcp_connections", 
				`node_netstat_Tcp_CurrEstab`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "tcp_retransmits", 
				`rate(node_netstat_Tcp_RetransSegs[5m])`)
			
			// CNI-specific metrics
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "cni_plugin_errors", 
				`rate(cni_operation_errors_total[5m])`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "pod_network_latency", 
				`histogram_quantile(0.95, rate(pod_network_latency_seconds_bucket[5m]))`)
		}
	}
}

// collectApplicationMetrics collects application performance metrics
func (mc *MetricsCollector) collectApplicationMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.appMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// HTTP metrics
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "http_requests_total", 
				`sum(rate(http_requests_total[5m])) by (service, method, status)`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "http_request_duration", 
				`histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "http_errors_5xx", 
				`sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "http_errors_4xx", 
				`sum(rate(http_requests_total{status=~"4.."}[5m])) by (service)`)
			
			// Database metrics
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "database_query_duration", 
				`histogram_quantile(0.95, rate(database_query_duration_seconds_bucket[5m]))`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "database_connections", 
				`database_connections_active`)
			
			// Application-specific metrics
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "app_error_rate", 
				`sum(rate(app_errors_total[5m])) by (service, error_type)`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "app_request_rate", 
				`sum(rate(app_requests_total[5m])) by (service)`)
			
			// Service mesh metrics (if available)
			mc.queryAndIngestMetric(ctx, MetricTypeApplication, "service_success_rate", 
				`sum(rate(istio_requests_total{response_code!~"5.."}[5m])) / sum(rate(istio_requests_total[5m]))`)
		}
	}
}

// collectInfrastructureMetrics collects infrastructure and node metrics
func (mc *MetricsCollector) collectInfrastructureMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.infraMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Node CPU metrics
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_cpu_usage", 
				`100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_cpu_steal", 
				`avg by (instance) (rate(node_cpu_seconds_total{mode="steal"}[5m])) * 100`)
			
			// Node memory metrics
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_memory_usage", 
				`(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_memory_pressure", 
				`rate(node_vmstat_pgmajfault[5m])`)
			
			// Node disk metrics
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_disk_usage", 
				`(node_filesystem_size_bytes - node_filesystem_avail_bytes) / node_filesystem_size_bytes * 100`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_disk_io_utilization", 
				`rate(node_disk_io_time_seconds_total[5m])`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_disk_iops", 
				`rate(node_disk_reads_completed_total[5m]) + rate(node_disk_writes_completed_total[5m])`)
			
			// Pod metrics
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "pod_cpu_usage", 
				`sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod, namespace)`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "pod_memory_usage", 
				`sum(container_memory_working_set_bytes{container!=""}) by (pod, namespace)`)
			
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "pod_restarts", 
				`rate(kube_pod_container_status_restarts_total[5m])`)
			
			// Node health
			mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_status", 
				`kube_node_status_condition{condition="Ready"}`)
		}
	}
}

// queryAndIngestMetric queries Prometheus and ingests the metric
func (mc *MetricsCollector) queryAndIngestMetric(ctx context.Context, metricType MetricType, name, query string) {
	result, warnings, err := mc.prometheusClient.Query(ctx, query, time.Now())
	if err != nil {
		fmt.Printf("Error querying Prometheus for %s: %v\n", name, err)
		return
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings for %s: %v\n", name, warnings)
	}

	// Process result based on type
	switch v := result.(type) {
	case model.Vector:
		for _, sample := range v {
			labels := make(map[string]string)
			for k, v := range sample.Metric {
				labels[string(k)] = string(v)
			}
			
			mc.correlationEngine.IngestMetric(
				metricType,
				name,
				float64(sample.Value),
				labels,
			)
		}
	case model.Matrix:
		// Handle range vectors
		for _, stream := range v {
			labels := make(map[string]string)
			for k, v := range stream.Metric {
				labels[string(k)] = string(v)
			}
			
			// Use the latest value
			if len(stream.Values) > 0 {
				latestValue := stream.Values[len(stream.Values)-1]
				mc.correlationEngine.IngestMetric(
					metricType,
					name,
					float64(latestValue.Value),
					labels,
				)
			}
		}
	}
}

// runCorrelationAnalysis periodically analyzes correlations
func (mc *MetricsCollector) runCorrelationAnalysis(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			correlations := mc.correlationEngine.AnalyzeCorrelations(ctx)
			
			// Log significant correlations
			for _, corr := range correlations {
				if corr.Confidence > 0.8 && corr.Impact.Severity != "low" {
					fmt.Printf("🔍 Detected %s correlation (%.2f confidence): %s <-> %s\n",
						corr.Impact.Severity,
						corr.Confidence,
						corr.PrimaryMetric.Name,
						corr.RelatedMetrics[0].Name,
					)
					
					if corr.RootCause != nil {
						fmt.Printf("   Root Cause: %s (%.0f%% confidence)\n",
							corr.RootCause.Cause,
							corr.RootCause.Confidence*100,
						)
					}
					
					if len(corr.Recommendations) > 0 {
						fmt.Printf("   Recommendation: %s\n", corr.Recommendations[0])
					}
				}
			}
		}
	}
}

// Example metric queries for specific scenarios

// GetPacketLossCorrelations finds correlations between packet loss and application errors
func (mc *MetricsCollector) GetPacketLossCorrelations(ctx context.Context) []Correlation {
	// Query for packet loss
	mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "packet_loss", 
		`rate(node_network_receive_drop_total[5m])`)
	
	// Query for HTTP 500 errors
	mc.queryAndIngestMetric(ctx, MetricTypeApplication, "http_500_errors", 
		`sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)`)
	
	// Analyze correlations
	correlations := mc.correlationEngine.AnalyzeCorrelations(ctx)
	
	// Filter for packet loss related correlations
	filtered := []Correlation{}
	for _, corr := range correlations {
		if corr.PrimaryMetric.Name == "packet_loss" || corr.RelatedMetrics[0].Name == "packet_loss" {
			filtered = append(filtered, corr)
		}
	}
	
	return filtered
}

// GetNodeHealthImpact analyzes how node health impacts application performance
func (mc *MetricsCollector) GetNodeHealthImpact(ctx context.Context) []Correlation {
	// Query node CPU usage
	mc.queryAndIngestMetric(ctx, MetricTypeInfrastructure, "node_cpu_high", 
		`avg by (instance) (rate(node_cpu_seconds_total{mode!="idle"}[5m])) * 100 > 80`)
	
	// Query application response time
	mc.queryAndIngestMetric(ctx, MetricTypeApplication, "response_time_p95", 
		`histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`)
	
	// Analyze correlations
	return mc.correlationEngine.AnalyzeCorrelations(ctx)
}

// GetCNIPerformanceImpact analyzes CNI plugin performance impact
func (mc *MetricsCollector) GetCNIPerformanceImpact(ctx context.Context) []Correlation {
	// Query CNI metrics
	mc.queryAndIngestMetric(ctx, MetricTypeNetwork, "cni_operation_duration", 
		`histogram_quantile(0.95, rate(cni_operation_duration_seconds_bucket[5m]))`)
	
	// Query pod startup time
	mc.queryAndIngestMetric(ctx, MetricTypeApplication, "pod_startup_duration", 
		`histogram_quantile(0.95, rate(pod_startup_duration_seconds_bucket[5m]))`)
	
	// Analyze correlations
	return mc.correlationEngine.AnalyzeCorrelations(ctx)
}
