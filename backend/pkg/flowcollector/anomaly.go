package flowcollector

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// AnomalyType represents types of flow anomalies
type AnomalyType string

const (
	AnomalyTrafficSpike      AnomalyType = "traffic_spike"
	AnomalyTrafficDrop       AnomalyType = "traffic_drop"
	AnomalyUnusualProtocol   AnomalyType = "unusual_protocol"
	AnomalyUnexpectedConn    AnomalyType = "unexpected_connection"
	AnomalyHighErrorRate     AnomalyType = "high_error_rate"
	AnomalyPortScan          AnomalyType = "port_scan"
	AnomalyDataExfiltration  AnomalyType = "data_exfiltration"
	AnomalyDNSAnomaly        AnomalyType = "dns_anomaly"
)

// Anomaly represents a detected network anomaly
type Anomaly struct {
	ID          string      `json:"id"`
	Type        AnomalyType `json:"type"`
	Severity    string      `json:"severity"` // critical, high, medium, low
	Title       string      `json:"title"`
	Description string      `json:"description"`
	SourcePod   string      `json:"source_pod"`
	DestPod     string      `json:"dest_pod,omitempty"`
	Evidence    Evidence    `json:"evidence"`
	DetectedAt  time.Time   `json:"detected_at"`
	Score       float64     `json:"score"` // 0-1, higher = more anomalous
}

// Evidence contains supporting data for an anomaly
type Evidence struct {
	CurrentValue  float64           `json:"current_value"`
	BaselineValue float64           `json:"baseline_value"`
	Threshold     float64           `json:"threshold"`
	Details       map[string]string `json:"details,omitempty"`
}

// AnomalyDetector detects anomalies in network flows
type AnomalyDetector struct {
	mu sync.RWMutex
	
	// Baselines for normal behavior
	trafficBaselines    map[string]*TrafficBaseline // key: source->dest
	protocolBaselines   map[string]map[string]int   // key: pod_id -> protocol counts
	connectionBaselines map[string][]string         // key: pod_id -> list of expected destinations
	
	// Detected anomalies
	anomalies []Anomaly
	maxAnomalies int
	
	// Configuration
	spikeThreshold      float64 // Multiplier for baseline to detect spike
	errorRateThreshold  float64 // Error rate % to trigger alert
	portScanThreshold   int     // Unique ports per minute to detect scan
	exfilThreshold      int64   // Bytes/sec threshold for exfiltration
}

// TrafficBaseline represents normal traffic patterns
type TrafficBaseline struct {
	AvgBytesPerSec   float64
	StdDevBytes      float64
	AvgPacketsPerSec float64
	StdDevPackets    float64
	AvgErrorRate     float64
	SampleCount      int
	LastUpdated      time.Time
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		trafficBaselines:    make(map[string]*TrafficBaseline),
		protocolBaselines:   make(map[string]map[string]int),
		connectionBaselines: make(map[string][]string),
		anomalies:           make([]Anomaly, 0),
		maxAnomalies:        1000,
		spikeThreshold:      3.0,  // 3x baseline = spike
		errorRateThreshold:  0.05, // 5% error rate
		portScanThreshold:   20,   // 20 unique ports in 1 min
		exfilThreshold:      10 * 1024 * 1024, // 10 MB/s
	}
}

// UpdateBaseline updates baseline metrics from flow metrics
func (ad *AnomalyDetector) UpdateBaseline(metrics map[string]*FlowMetric) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	for key, metric := range metrics {
		baseline, exists := ad.trafficBaselines[key]
		if !exists {
			baseline = &TrafficBaseline{
				LastUpdated: time.Now(),
			}
			ad.trafficBaselines[key] = baseline
		}
		
		// Update baseline using exponential moving average
		alpha := 0.3 // Weight for new sample
		baseline.AvgBytesPerSec = alpha*metric.BytesPerSec + (1-alpha)*baseline.AvgBytesPerSec
		baseline.AvgPacketsPerSec = alpha*metric.PacketsPerSec + (1-alpha)*baseline.AvgPacketsPerSec
		baseline.AvgErrorRate = alpha*metric.ErrorRate + (1-alpha)*baseline.AvgErrorRate
		baseline.SampleCount++
		baseline.LastUpdated = time.Now()
		
		// Calculate standard deviation (simplified running stddev)
		if baseline.SampleCount > 1 {
			byteDiff := metric.BytesPerSec - baseline.AvgBytesPerSec
			baseline.StdDevBytes = math.Sqrt(
				(baseline.StdDevBytes*baseline.StdDevBytes*float64(baseline.SampleCount-1) + 
				byteDiff*byteDiff) / float64(baseline.SampleCount),
			)
			
			packetDiff := metric.PacketsPerSec - baseline.AvgPacketsPerSec
			baseline.StdDevPackets = math.Sqrt(
				(baseline.StdDevPackets*baseline.StdDevPackets*float64(baseline.SampleCount-1) + 
				packetDiff*packetDiff) / float64(baseline.SampleCount),
			)
		}
	}
}

// AnalyzeFlows analyzes flows for anomalies
func (ad *AnomalyDetector) AnalyzeFlows(flows []*Flow, metrics map[string]*FlowMetric) []Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	newAnomalies := make([]Anomaly, 0)
	
	// 1. Detect traffic spikes/drops
	newAnomalies = append(newAnomalies, ad.detectTrafficAnomalies(metrics)...)
	
	// 2. Detect unusual protocols
	newAnomalies = append(newAnomalies, ad.detectUnusualProtocols(flows)...)
	
	// 3. Detect unexpected connections
	newAnomalies = append(newAnomalies, ad.detectUnexpectedConnections(flows)...)
	
	// 4. Detect high error rates
	newAnomalies = append(newAnomalies, ad.detectHighErrorRate(metrics)...)
	
	// TODO: Re-enable after fixing type compatibility
	// 5. Detect port scanning
	//newAnomalies = append(newAnomalies, ad.detectPortScanning(flows)...)
	
	// 6. Detect potential data exfiltration
	//newAnomalies = append(newAnomalies, ad.detectDataExfiltration(metrics)...)
	
	// 7. Detect DNS anomalies
	//newAnomalies = append(newAnomalies, ad.detectDNSAnomalies(flows)...)
	
	// Store anomalies
	ad.anomalies = append(ad.anomalies, newAnomalies...)
	if len(ad.anomalies) > ad.maxAnomalies {
		ad.anomalies = ad.anomalies[len(ad.anomalies)-ad.maxAnomalies:]
	}
	
	return newAnomalies
}

// detectTrafficAnomalies detects traffic spikes and drops
func (ad *AnomalyDetector) detectTrafficAnomalies(metrics map[string]*FlowMetric) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	for key, metric := range metrics {
		baseline, exists := ad.trafficBaselines[key]
		
		// Need sufficient baseline data
		if !exists || baseline.SampleCount < 10 {
			continue
		}
		
		sourcePod := fmt.Sprintf("%s/%s", metric.SourceNamespace, metric.SourcePod)
		destPod := fmt.Sprintf("%s/%s", metric.DestNamespace, metric.DestPod)
		
		// Check for traffic spike
		threshold := baseline.AvgBytesPerSec + (ad.spikeThreshold * baseline.StdDevBytes)
		if metric.BytesPerSec > threshold && baseline.AvgBytesPerSec > 0 {
			score := (metric.BytesPerSec - baseline.AvgBytesPerSec) / baseline.AvgBytesPerSec
			anomaly := Anomaly{
				ID:         fmt.Sprintf("spike-%s-%d", key, time.Now().Unix()),
				Type:       AnomalyTrafficSpike,
				Severity:   ad.calculateSeverity(score),
				Title:      "Traffic Spike Detected",
				Description: fmt.Sprintf("Traffic from %s to %s is %.1fx higher than baseline", 
					sourcePod, destPod, metric.BytesPerSec/baseline.AvgBytesPerSec),
				SourcePod:  sourcePod,
				DestPod:    destPod,
				Evidence: Evidence{
					CurrentValue:  metric.BytesPerSec,
					BaselineValue: baseline.AvgBytesPerSec,
					Threshold:     threshold,
					Details: map[string]string{
						"baseline_stddev": fmt.Sprintf("%.2f", baseline.StdDevBytes),
						"multiplier":      fmt.Sprintf("%.1fx", metric.BytesPerSec/baseline.AvgBytesPerSec),
					},
				},
				DetectedAt: time.Now(),
				Score:      math.Min(score/10, 1.0),
			}
			anomalies = append(anomalies, anomaly)
		}
		
		// Check for traffic drop
		if baseline.AvgBytesPerSec > 1000 && metric.BytesPerSec < baseline.AvgBytesPerSec*0.2 {
			anomaly := Anomaly{
				ID:         fmt.Sprintf("drop-%s-%d", key, time.Now().Unix()),
				Type:       AnomalyTrafficDrop,
				Severity:   "medium",
				Title:      "Traffic Drop Detected",
				Description: fmt.Sprintf("Traffic from %s to %s has dropped significantly", 
					sourcePod, destPod),
				SourcePod:  sourcePod,
				DestPod:    destPod,
				Evidence: Evidence{
					CurrentValue:  metric.BytesPerSec,
					BaselineValue: baseline.AvgBytesPerSec,
					Details: map[string]string{
						"drop_percentage": fmt.Sprintf("%.1f%%", 
							(1-metric.BytesPerSec/baseline.AvgBytesPerSec)*100),
					},
				},
				DetectedAt: time.Now(),
				Score:      0.6,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// detectUnusualProtocols detects protocols not normally used by a pod
func (ad *AnomalyDetector) detectUnusualProtocols(flows []*Flow) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	// Count protocols per pod
	recentProtocols := make(map[string]map[string]int)
	for _, flow := range flows {
		if time.Since(flow.Timestamp) > 5*time.Minute {
			continue
		}
		
		if _, exists := recentProtocols[flow.SourcePod]; !exists {
			recentProtocols[flow.SourcePod] = make(map[string]int)
		}
		recentProtocols[flow.SourcePod][flow.Protocol]++
	}
	
	// Check against baseline
	for podID, protocols := range recentProtocols {
		baseline, exists := ad.protocolBaselines[podID]
		if !exists {
			// Create baseline for new pod
			ad.protocolBaselines[podID] = protocols
			continue
		}
		
		// Check for new protocols
		for protocol := range protocols {
			if _, expected := baseline[protocol]; !expected {
				anomaly := Anomaly{
					ID:          fmt.Sprintf("proto-%s-%s-%d", podID, protocol, time.Now().Unix()),
					Type:        AnomalyUnusualProtocol,
					Severity:    "medium",
					Title:       "Unusual Protocol Detected",
					Description: fmt.Sprintf("Pod %s is using protocol %s which is not in baseline", podID, protocol),
					SourcePod:   podID,
					Evidence: Evidence{
						Details: map[string]string{
							"protocol": string(protocol),
							"count":    fmt.Sprintf("%d", protocols[protocol]),
						},
					},
					DetectedAt: time.Now(),
					Score:      0.5,
				}
				anomalies = append(anomalies, anomaly)
			}
		}
		
		// Update baseline
		for protocol, count := range protocols {
			baseline[protocol] += count
		}
	}
	
	return anomalies
}

// detectUnexpectedConnections detects connections to new destinations
func (ad *AnomalyDetector) detectUnexpectedConnections(flows []*Flow) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	// Track recent connections
	recentConns := make(map[string]map[string]bool)
	for _, flow := range flows {
		if time.Since(flow.Timestamp) > 10*time.Minute {
			continue
		}
		
		if _, exists := recentConns[flow.SourcePod]; !exists {
			recentConns[flow.SourcePod] = make(map[string]bool)
		}
		recentConns[flow.SourcePod][flow.DestPod] = true
	}
	
	// Check against baseline
	for sourcePod, destinations := range recentConns {
		baseline, exists := ad.connectionBaselines[sourcePod]
		if !exists {
			// Create baseline
			ad.connectionBaselines[sourcePod] = make([]string, 0)
			for dest := range destinations {
				ad.connectionBaselines[sourcePod] = append(ad.connectionBaselines[sourcePod], dest)
			}
			continue
		}
		
		// Check for unexpected destinations
		baselineMap := make(map[string]bool)
		for _, dest := range baseline {
			baselineMap[dest] = true
		}
		
		for dest := range destinations {
			if _, expected := baselineMap[dest]; !expected && dest != "" {
				anomaly := Anomaly{
					ID:          fmt.Sprintf("conn-%s-%s-%d", sourcePod, dest, time.Now().Unix()),
					Type:        AnomalyUnexpectedConn,
					Severity:    "low",
					Title:       "Unexpected Connection",
					Description: fmt.Sprintf("Pod %s connected to unexpected destination %s", sourcePod, dest),
					SourcePod:   sourcePod,
					DestPod:     dest,
					Evidence: Evidence{
						Details: map[string]string{
							"new_destination": dest,
						},
					},
					DetectedAt: time.Now(),
					Score:      0.4,
				}
				anomalies = append(anomalies, anomaly)
				
				// Add to baseline
				ad.connectionBaselines[sourcePod] = append(ad.connectionBaselines[sourcePod], dest)
			}
		}
	}
	
	return anomalies
}

// detectHighErrorRate detects elevated error rates
func (ad *AnomalyDetector) detectHighErrorRate(metrics map[string]*FlowMetric) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	for key, metric := range metrics {
		if metric.ErrorRate > ad.errorRateThreshold {
			severity := "medium"
			if metric.ErrorRate > 0.1 {
				severity = "high"
			}
			if metric.ErrorRate > 0.25 {
				severity = "critical"
			}
			
			sourcePod := fmt.Sprintf("%s/%s", metric.SourceNamespace, metric.SourcePod)
			destPod := fmt.Sprintf("%s/%s", metric.DestNamespace, metric.DestPod)
			
			anomaly := Anomaly{
				ID:          fmt.Sprintf("error-%s-%d", key, time.Now().Unix()),
				Type:        AnomalyHighErrorRate,
				Severity:    severity,
				Title:       "High Error Rate Detected",
				Description: fmt.Sprintf("Connection from %s to %s has %.1f%% error rate", 
					sourcePod, destPod, metric.ErrorRate*100),
				SourcePod:   sourcePod,
				DestPod:     destPod,
				Evidence: Evidence{
					CurrentValue: metric.ErrorRate * 100,
					Threshold:    ad.errorRateThreshold * 100,
					Details: map[string]string{
						"error_percentage": fmt.Sprintf("%.1f%%", metric.ErrorRate*100),
					},
				},
				DetectedAt: time.Now(),
				Score:      metric.ErrorRate,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// detectPortScanning detects port scanning behavior
func (ad *AnomalyDetector) detectPortScanning(flows []NetworkFlow) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	// Track unique dest ports per source in last minute
	portsBySource := make(map[string]map[uint32]bool)
	cutoff := time.Now().Add(-1 * time.Minute)
	
	for _, flow := range flows {
		if flow.Timestamp.Before(cutoff) {
			continue
		}
		
		if _, exists := portsBySource[flow.SourcePod]; !exists {
			portsBySource[flow.SourcePod] = make(map[uint32]bool)
		}
		portsBySource[flow.SourcePod][flow.DestPort] = true
	}
	
	// Check for port scans
	for sourcePod, ports := range portsBySource {
		if len(ports) > ad.portScanThreshold {
			anomaly := Anomaly{
				ID:          fmt.Sprintf("portscan-%s-%d", sourcePod, time.Now().Unix()),
				Type:        AnomalyPortScan,
				Severity:    "high",
				Title:       "Potential Port Scan Detected",
				Description: fmt.Sprintf("Pod %s connected to %d unique ports in 1 minute", sourcePod, len(ports)),
				SourcePod:   sourcePod,
				Evidence: Evidence{
					CurrentValue: float64(len(ports)),
					Threshold:    float64(ad.portScanThreshold),
					Details: map[string]string{
						"unique_ports": fmt.Sprintf("%d", len(ports)),
						"time_window":  "1 minute",
					},
				},
				DetectedAt: time.Now(),
				Score:      0.8,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// detectDataExfiltration detects potential data exfiltration
func (ad *AnomalyDetector) detectDataExfiltration(metrics []FlowMetrics) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	for _, metric := range metrics {
		// Check for high outbound traffic
		if metric.BytesPerSec > float64(ad.exfilThreshold) {
			anomaly := Anomaly{
				ID:          fmt.Sprintf("exfil-%s-%s-%d", metric.SourceID, metric.DestID, time.Now().Unix()),
				Type:        AnomalyDataExfiltration,
				Severity:    "critical",
				Title:       "Potential Data Exfiltration",
				Description: fmt.Sprintf("Very high outbound traffic from %s to %s: %.2f MB/s", 
					metric.SourceID, metric.DestID, metric.BytesPerSec/1024/1024),
				SourcePod:   metric.SourceID,
				DestPod:     metric.DestID,
				Evidence: Evidence{
					CurrentValue: metric.BytesPerSec,
					Threshold:    float64(ad.exfilThreshold),
					Details: map[string]string{
						"bandwidth_mbps": fmt.Sprintf("%.2f", metric.BytesPerSec/1024/1024),
					},
				},
				DetectedAt: time.Now(),
				Score:      0.9,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// detectDNSAnomalies detects DNS-related anomalies
func (ad *AnomalyDetector) detectDNSAnomalies(flows []NetworkFlow) []Anomaly {
	anomalies := make([]Anomaly, 0)
	
	// Track DNS queries per pod
	dnsQueriesByPod := make(map[string]int)
	cutoff := time.Now().Add(-1 * time.Minute)
	
	for _, flow := range flows {
		if flow.Timestamp.Before(cutoff) || flow.L7Protocol != string(ProtocolDNS) {
			continue
		}
		
		dnsQueriesByPod[flow.SourcePod]++
	}
	
	// Check for excessive DNS queries
	for podID, count := range dnsQueriesByPod {
		if count > 100 { // More than 100 DNS queries per minute
			anomaly := Anomaly{
				ID:          fmt.Sprintf("dns-%s-%d", podID, time.Now().Unix()),
				Type:        AnomalyDNSAnomaly,
				Severity:    "medium",
				Title:       "Excessive DNS Queries",
				Description: fmt.Sprintf("Pod %s made %d DNS queries in 1 minute", podID, count),
				SourcePod:   podID,
				Evidence: Evidence{
					CurrentValue: float64(count),
					Threshold:    100,
					Details: map[string]string{
						"query_count": fmt.Sprintf("%d", count),
						"time_window": "1 minute",
					},
				},
				DetectedAt: time.Now(),
				Score:      0.6,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// calculateSeverity determines severity based on anomaly score
func (ad *AnomalyDetector) calculateSeverity(score float64) string {
	if score >= 5.0 {
		return "critical"
	} else if score >= 3.0 {
		return "high"
	} else if score >= 1.5 {
		return "medium"
	}
	return "low"
}

// GetAnomalies returns detected anomalies
func (ad *AnomalyDetector) GetAnomalies(limit int) []Anomaly {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	
	if limit <= 0 || limit > len(ad.anomalies) {
		limit = len(ad.anomalies)
	}
	
	// Return most recent anomalies
	start := len(ad.anomalies) - limit
	anomalies := make([]Anomaly, limit)
	copy(anomalies, ad.anomalies[start:])
	
	return anomalies
}

// GetAnomaliesBySeverity returns anomalies filtered by severity
func (ad *AnomalyDetector) GetAnomaliesBySeverity(severity string) []Anomaly {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	
	anomalies := make([]Anomaly, 0)
	for _, anomaly := range ad.anomalies {
		if anomaly.Severity == severity {
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}
