package correlation

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// CorrelationEngine correlates network, application, and infrastructure metrics
type CorrelationEngine struct {
	networkMetrics      *MetricStore
	applicationMetrics  *MetricStore
	infrastructureMetrics *MetricStore
	correlations        map[string]*Correlation
	mu                  sync.RWMutex
	
	// Thresholds for correlation detection
	correlationThreshold float64
	timeWindow          time.Duration
}

// MetricStore holds time-series metrics
type MetricStore struct {
	metrics map[string]*TimeSeries
	mu      sync.RWMutex
}

// TimeSeries represents a time-series metric
type TimeSeries struct {
	Name       string
	Labels     map[string]string
	DataPoints []DataPoint
	Type       MetricType
}

// DataPoint represents a single metric value at a specific time
type DataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// MetricType defines the category of metric
type MetricType string

const (
	MetricTypeNetwork        MetricType = "network"
	MetricTypeApplication    MetricType = "application"
	MetricTypeInfrastructure MetricType = "infrastructure"
)

// Correlation represents a detected correlation between metrics
type Correlation struct {
	ID              string                 `json:"id"`
	Type            CorrelationType        `json:"type"`
	Confidence      float64                `json:"confidence"`
	Strength        float64                `json:"strength"`
	TimeDetected    time.Time              `json:"time_detected"`
	PrimaryMetric   MetricReference        `json:"primary_metric"`
	RelatedMetrics  []MetricReference      `json:"related_metrics"`
	Impact          CorrelationImpact      `json:"impact"`
	RootCause       *RootCauseAnalysis     `json:"root_cause"`
	Recommendations []string               `json:"recommendations"`
	Visualization   CorrelationVisualization `json:"visualization"`
}

// CorrelationType defines types of correlations
type CorrelationType string

const (
	CorrelationTypePositive    CorrelationType = "positive"    // Metrics move together
	CorrelationTypeNegative    CorrelationType = "negative"    // Metrics move inversely
	CorrelationTypeCausation   CorrelationType = "causation"   // One causes the other
	CorrelationTypeCoincidental CorrelationType = "coincidental" // Happen together but not related
)

// MetricReference points to a specific metric
type MetricReference struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Source      string            `json:"source"`
	Labels      map[string]string `json:"labels"`
	CurrentValue float64          `json:"current_value"`
	Baseline    float64           `json:"baseline"`
	Deviation   float64           `json:"deviation"`
}

// CorrelationImpact describes the business impact
type CorrelationImpact struct {
	Severity        string   `json:"severity"`
	AffectedServices []string `json:"affected_services"`
	AffectedUsers   int      `json:"affected_users"`
	ErrorRate       float64  `json:"error_rate"`
	Latency         float64  `json:"latency_ms"`
	Availability    float64  `json:"availability_pct"`
	BusinessImpact  string   `json:"business_impact"`
}

// RootCauseAnalysis identifies the root cause
type RootCauseAnalysis struct {
	Cause           string               `json:"cause"`
	Confidence      float64              `json:"confidence"`
	Timeline        []CausalEvent        `json:"timeline"`
	AffectedLayer   string               `json:"affected_layer"`
	PropagationPath []PropagationStep    `json:"propagation_path"`
}

// CausalEvent represents an event in the causal chain
type CausalEvent struct {
	Time        time.Time `json:"time"`
	Event       string    `json:"event"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Impact      string    `json:"impact"`
}

// PropagationStep shows how issues propagate
type PropagationStep struct {
	Step        int       `json:"step"`
	Layer       string    `json:"layer"`
	Component   string    `json:"component"`
	Issue       string    `json:"issue"`
	Time        time.Time `json:"time"`
	Duration    float64   `json:"duration_ms"`
}

// CorrelationVisualization provides data for UI visualization
type CorrelationVisualization struct {
	GraphType   string              `json:"graph_type"`
	Nodes       []VisualizationNode `json:"nodes"`
	Edges       []VisualizationEdge `json:"edges"`
	TimeSeries  []TimeSeriesData    `json:"time_series"`
	Annotations []Annotation        `json:"annotations"`
}

// VisualizationNode represents a metric or component
type VisualizationNode struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"`
	Status   string            `json:"status"`
	Metrics  map[string]float64 `json:"metrics"`
	Position Position          `json:"position"`
}

// VisualizationEdge represents a relationship
type VisualizationEdge struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	Type       string  `json:"type"`
	Strength   float64 `json:"strength"`
	Direction  string  `json:"direction"`
	Label      string  `json:"label"`
}

// TimeSeriesData for charting
type TimeSeriesData struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Color  string      `json:"color"`
	Points []ChartPoint `json:"points"`
}

// ChartPoint represents a point on a chart
type ChartPoint struct {
	X     time.Time `json:"x"`
	Y     float64   `json:"y"`
	Label string    `json:"label,omitempty"`
}

// Annotation marks significant events
type Annotation struct {
	Time        time.Time `json:"time"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
}

// Position for graph layout
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NewCorrelationEngine creates a new correlation engine
func NewCorrelationEngine(timeWindow time.Duration) *CorrelationEngine {
	return &CorrelationEngine{
		networkMetrics:        NewMetricStore(),
		applicationMetrics:    NewMetricStore(),
		infrastructureMetrics: NewMetricStore(),
		correlations:          make(map[string]*Correlation),
		correlationThreshold:  0.7,
		timeWindow:           timeWindow,
	}
}

// NewMetricStore creates a new metric store
func NewMetricStore() *MetricStore {
	return &MetricStore{
		metrics: make(map[string]*TimeSeries),
	}
}

// IngestMetric adds a metric to the store
func (ce *CorrelationEngine) IngestMetric(metricType MetricType, name string, value float64, labels map[string]string) {
	var store *MetricStore
	
	switch metricType {
	case MetricTypeNetwork:
		store = ce.networkMetrics
	case MetricTypeApplication:
		store = ce.applicationMetrics
	case MetricTypeInfrastructure:
		store = ce.infrastructureMetrics
	default:
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	key := fmt.Sprintf("%s_%s", name, labelsToKey(labels))
	
	if _, exists := store.metrics[key]; !exists {
		store.metrics[key] = &TimeSeries{
			Name:       name,
			Labels:     labels,
			DataPoints: make([]DataPoint, 0),
			Type:       metricType,
		}
	}

	dataPoint := DataPoint{
		Timestamp: time.Now(),
		Value:     value,
		Labels:    labels,
	}

	store.metrics[key].DataPoints = append(store.metrics[key].DataPoints, dataPoint)
	
	// Keep only data within time window
	cutoff := time.Now().Add(-ce.timeWindow)
	store.metrics[key].DataPoints = filterDataPoints(store.metrics[key].DataPoints, cutoff)
}

// AnalyzeCorrelations finds correlations across all metrics
func (ce *CorrelationEngine) AnalyzeCorrelations(ctx context.Context) []Correlation {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	correlations := []Correlation{}

	// Get all time series
	networkSeries := ce.networkMetrics.GetAllSeries()
	appSeries := ce.applicationMetrics.GetAllSeries()
	infraSeries := ce.infrastructureMetrics.GetAllSeries()

	// Find correlations between network and application metrics
	for _, netMetric := range networkSeries {
		for _, appMetric := range appSeries {
			if corr := ce.calculateCorrelation(netMetric, appMetric); corr != nil {
				correlations = append(correlations, *corr)
			}
		}
	}

	// Find correlations between network and infrastructure metrics
	for _, netMetric := range networkSeries {
		for _, infraMetric := range infraSeries {
			if corr := ce.calculateCorrelation(netMetric, infraMetric); corr != nil {
				correlations = append(correlations, *corr)
			}
		}
	}

	// Find correlations between application and infrastructure metrics
	for _, appMetric := range appSeries {
		for _, infraMetric := range infraSeries {
			if corr := ce.calculateCorrelation(appMetric, infraMetric); corr != nil {
				correlations = append(correlations, *corr)
			}
		}
	}

	// Store correlations
	for _, corr := range correlations {
		ce.correlations[corr.ID] = &corr
	}

	return correlations
}

// calculateCorrelation computes Pearson correlation between two time series
func (ce *CorrelationEngine) calculateCorrelation(ts1, ts2 *TimeSeries) *Correlation {
	// Align time series
	aligned1, aligned2 := alignTimeSeries(ts1, ts2)
	
	if len(aligned1) < 2 {
		return nil
	}

	// Calculate Pearson correlation coefficient
	coeff := pearsonCorrelation(aligned1, aligned2)
	
	if math.Abs(coeff) < ce.correlationThreshold {
		return nil
	}

	// Determine correlation type
	corrType := CorrelationTypePositive
	if coeff < 0 {
		corrType = CorrelationTypeNegative
	}

	// Check for causation using Granger causality test (simplified)
	if ce.testCausation(ts1, ts2) {
		corrType = CorrelationTypeCausation
	}

	correlation := &Correlation{
		ID:           fmt.Sprintf("corr_%s_%s_%d", ts1.Name, ts2.Name, time.Now().Unix()),
		Type:         corrType,
		Confidence:   math.Abs(coeff),
		Strength:     math.Abs(coeff),
		TimeDetected: time.Now(),
		PrimaryMetric: MetricReference{
			Name:         ts1.Name,
			Type:         ts1.Type,
			Labels:       ts1.Labels,
			CurrentValue: aligned1[len(aligned1)-1],
			Baseline:     average(aligned1),
			Deviation:    (aligned1[len(aligned1)-1] - average(aligned1)) / average(aligned1) * 100,
		},
		RelatedMetrics: []MetricReference{
			{
				Name:         ts2.Name,
				Type:         ts2.Type,
				Labels:       ts2.Labels,
				CurrentValue: aligned2[len(aligned2)-1],
				Baseline:     average(aligned2),
				Deviation:    (aligned2[len(aligned2)-1] - average(aligned2)) / average(aligned2) * 100,
			},
		},
	}

	// Analyze impact
	correlation.Impact = ce.analyzeImpact(ts1, ts2, aligned1, aligned2)
	
	// Perform root cause analysis
	correlation.RootCause = ce.performRootCauseAnalysis(ts1, ts2, aligned1, aligned2)
	
	// Generate recommendations
	correlation.Recommendations = ce.generateRecommendations(correlation)
	
	// Create visualization data
	correlation.Visualization = ce.createVisualization(ts1, ts2, correlation)

	return correlation
}

// analyzeImpact determines the business impact
func (ce *CorrelationEngine) analyzeImpact(ts1, ts2 *TimeSeries, values1, values2 []float64) CorrelationImpact {
	impact := CorrelationImpact{
		Severity: "low",
	}

	// Check if metrics indicate errors
	errorRate := 0.0
	if ts1.Name == "http_errors" || ts2.Name == "http_errors" {
		errorRate = values2[len(values2)-1]
		impact.ErrorRate = errorRate
	}

	// Check latency
	if ts1.Name == "response_time" || ts2.Name == "response_time" {
		latency := values2[len(values2)-1]
		impact.Latency = latency
		
		if latency > 1000 {
			impact.Severity = "high"
		} else if latency > 500 {
			impact.Severity = "medium"
		}
	}

	// Check packet loss
	if ts1.Name == "packet_loss" || ts2.Name == "packet_loss" {
		loss := values1[len(values1)-1]
		if loss > 5 {
			impact.Severity = "critical"
		} else if loss > 1 {
			impact.Severity = "high"
		}
	}

	// Extract affected services from labels
	if service, ok := ts1.Labels["service"]; ok {
		impact.AffectedServices = append(impact.AffectedServices, service)
	}
	if service, ok := ts2.Labels["service"]; ok {
		impact.AffectedServices = append(impact.AffectedServices, service)
	}

	// Estimate affected users (simplified)
	impact.AffectedUsers = ce.estimateAffectedUsers(impact.AffectedServices, errorRate)
	
	// Calculate availability
	impact.Availability = 100.0 - errorRate
	
	// Describe business impact
	impact.BusinessImpact = ce.describeBusinessImpact(impact)

	return impact
}

// performRootCauseAnalysis identifies the root cause
func (ce *CorrelationEngine) performRootCauseAnalysis(ts1, ts2 *TimeSeries, values1, values2 []float64) *RootCauseAnalysis {
	rca := &RootCauseAnalysis{
		Timeline:        []CausalEvent{},
		PropagationPath: []PropagationStep{},
	}

	// Determine which metric occurred first
	ts1Start := ts1.DataPoints[0].Timestamp
	ts2Start := ts2.DataPoints[0].Timestamp

	var primaryMetric, secondaryMetric *TimeSeries
	var primaryValues []float64

	if ts1Start.Before(ts2Start) {
		primaryMetric = ts1
		secondaryMetric = ts2
		primaryValues = values1
	} else {
		primaryMetric = ts2
		secondaryMetric = ts1
		primaryValues = values2
	}

	// Identify the root cause
	if primaryMetric.Type == MetricTypeInfrastructure {
		rca.Cause = fmt.Sprintf("Infrastructure issue: %s", primaryMetric.Name)
		rca.AffectedLayer = "Infrastructure"
		rca.Confidence = 0.9
	} else if primaryMetric.Type == MetricTypeNetwork {
		rca.Cause = fmt.Sprintf("Network issue: %s", primaryMetric.Name)
		rca.AffectedLayer = "Network"
		rca.Confidence = 0.85
	} else {
		rca.Cause = fmt.Sprintf("Application issue: %s", primaryMetric.Name)
		rca.AffectedLayer = "Application"
		rca.Confidence = 0.75
	}

	// Build timeline
	for i := 0; i < len(primaryValues); i++ {
		if i < len(primaryMetric.DataPoints) {
			event := CausalEvent{
				Time:   primaryMetric.DataPoints[i].Timestamp,
				Event:  fmt.Sprintf("%s changed to %.2f", primaryMetric.Name, primaryValues[i]),
				Metric: primaryMetric.Name,
				Value:  primaryValues[i],
				Impact: ce.describeImpact(primaryValues[i], primaryMetric.Name),
			}
			rca.Timeline = append(rca.Timeline, event)
		}
	}

	// Build propagation path
	rca.PropagationPath = ce.buildPropagationPath(primaryMetric, secondaryMetric)

	return rca
}

// buildPropagationPath shows how issues propagate through layers
func (ce *CorrelationEngine) buildPropagationPath(primary, secondary *TimeSeries) []PropagationStep {
	path := []PropagationStep{}
	
	step := 1
	baseTime := time.Now().Add(-ce.timeWindow)

	// Infrastructure -> Network -> Application
	if primary.Type == MetricTypeInfrastructure {
		path = append(path, PropagationStep{
			Step:      step,
			Layer:     "Infrastructure",
			Component: getComponent(primary.Labels),
			Issue:     fmt.Sprintf("%s degradation", primary.Name),
			Time:      baseTime,
			Duration:  0,
		})
		step++

		if secondary.Type == MetricTypeNetwork {
			path = append(path, PropagationStep{
				Step:      step,
				Layer:     "Network",
				Component: getComponent(secondary.Labels),
				Issue:     fmt.Sprintf("%s affected", secondary.Name),
				Time:      baseTime.Add(5 * time.Second),
				Duration:  5000,
			})
		} else if secondary.Type == MetricTypeApplication {
			path = append(path, PropagationStep{
				Step:      step,
				Layer:     "Network",
				Component: "CNI",
				Issue:     "Packet loss",
				Time:      baseTime.Add(5 * time.Second),
				Duration:  5000,
			})
			step++
			
			path = append(path, PropagationStep{
				Step:      step,
				Layer:     "Application",
				Component: getComponent(secondary.Labels),
				Issue:     fmt.Sprintf("%s errors", secondary.Name),
				Time:      baseTime.Add(15 * time.Second),
				Duration:  10000,
			})
		}
	}

	return path
}

// generateRecommendations provides actionable recommendations
func (ce *CorrelationEngine) generateRecommendations(corr *Correlation) []string {
	recommendations := []string{}

	if corr.RootCause != nil {
		switch corr.RootCause.AffectedLayer {
		case "Infrastructure":
			recommendations = append(recommendations,
				"Check node resource utilization (CPU, memory, disk I/O)",
				"Review node logs for hardware or kernel issues",
				"Consider node drain and restart if issues persist",
				"Scale cluster or add more nodes if at capacity",
			)
		case "Network":
			recommendations = append(recommendations,
				"Inspect CNI plugin logs and configuration",
				"Check network policies for misconfigurations",
				"Verify pod network connectivity with tcpdump",
				"Review firewall rules and security groups",
				"Check for DNS resolution issues",
			)
		case "Application":
			recommendations = append(recommendations,
				"Review application logs for errors and exceptions",
				"Check service endpoints and health checks",
				"Analyze APM traces for slow transactions",
				"Review recent deployments for breaking changes",
				"Check database connections and query performance",
			)
		}
	}

	// Add specific recommendations based on metrics
	if corr.Impact.ErrorRate > 5 {
		recommendations = append(recommendations,
			fmt.Sprintf("Error rate is %.2f%% - investigate error logs immediately", corr.Impact.ErrorRate),
		)
	}

	if corr.Impact.Latency > 1000 {
		recommendations = append(recommendations,
			fmt.Sprintf("Latency is %.0fms - review slow queries and optimize critical paths", corr.Impact.Latency),
		)
	}

	return recommendations
}

// createVisualization generates visualization data
func (ce *CorrelationEngine) createVisualization(ts1, ts2 *TimeSeries, corr *Correlation) CorrelationVisualization {
	viz := CorrelationVisualization{
		GraphType: "correlation_graph",
		Nodes:     []VisualizationNode{},
		Edges:     []VisualizationEdge{},
		TimeSeries: []TimeSeriesData{},
		Annotations: []Annotation{},
	}

	// Create nodes
	viz.Nodes = append(viz.Nodes,
		VisualizationNode{
			ID:    ts1.Name,
			Label: ts1.Name,
			Type:  string(ts1.Type),
			Status: ce.getMetricStatus(corr.PrimaryMetric),
			Position: Position{X: 100, Y: 100},
		},
		VisualizationNode{
			ID:    ts2.Name,
			Label: ts2.Name,
			Type:  string(ts2.Type),
			Status: ce.getMetricStatus(corr.RelatedMetrics[0]),
			Position: Position{X: 300, Y: 100},
		},
	)

	// Create edge
	viz.Edges = append(viz.Edges, VisualizationEdge{
		Source:    ts1.Name,
		Target:    ts2.Name,
		Type:      string(corr.Type),
		Strength:  corr.Strength,
		Direction: "forward",
		Label:     fmt.Sprintf("%.2f correlation", corr.Strength),
	})

	// Create time series data
	viz.TimeSeries = append(viz.TimeSeries,
		TimeSeriesData{
			Name:   ts1.Name,
			Type:   string(ts1.Type),
			Color:  ce.getMetricColor(ts1.Type),
			Points: ce.toChartPoints(ts1.DataPoints),
		},
		TimeSeriesData{
			Name:   ts2.Name,
			Type:   string(ts2.Type),
			Color:  ce.getMetricColor(ts2.Type),
			Points: ce.toChartPoints(ts2.DataPoints),
		},
	)

	// Add annotations for significant events
	if corr.RootCause != nil {
		for _, event := range corr.RootCause.Timeline {
			viz.Annotations = append(viz.Annotations, Annotation{
				Time:        event.Time,
				Label:       event.Event,
				Description: event.Impact,
				Severity:    corr.Impact.Severity,
			})
		}
	}

	return viz
}

// Helper functions

func labelsToKey(labels map[string]string) string {
	key := ""
	for k, v := range labels {
		key += fmt.Sprintf("%s=%s,", k, v)
	}
	return key
}

func filterDataPoints(points []DataPoint, cutoff time.Time) []DataPoint {
	filtered := []DataPoint{}
	for _, p := range points {
		if p.Timestamp.After(cutoff) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (ms *MetricStore) GetAllSeries() []*TimeSeries {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	series := make([]*TimeSeries, 0, len(ms.metrics))
	for _, ts := range ms.metrics {
		series = append(series, ts)
	}
	return series
}

func alignTimeSeries(ts1, ts2 *TimeSeries) ([]float64, []float64) {
	// Simple alignment by time windows
	aligned1 := []float64{}
	aligned2 := []float64{}

	for _, dp1 := range ts1.DataPoints {
		// Find closest point in ts2
		for _, dp2 := range ts2.DataPoints {
			if math.Abs(dp1.Timestamp.Sub(dp2.Timestamp).Seconds()) < 60 { // Within 1 minute
				aligned1 = append(aligned1, dp1.Value)
				aligned2 = append(aligned2, dp2.Value)
				break
			}
		}
	}

	return aligned1, aligned2
}

func pearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}

	n := float64(len(x))
	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func (ce *CorrelationEngine) testCausation(ts1, ts2 *TimeSeries) bool {
	// Simplified Granger causality test
	// Check if ts1 values predict ts2 values
	if len(ts1.DataPoints) < 3 || len(ts2.DataPoints) < 3 {
		return false
	}

	// Check if ts1 peaks occur before ts2 peaks
	ts1Peaks := ce.findPeaks(ts1.DataPoints)
	ts2Peaks := ce.findPeaks(ts2.DataPoints)

	if len(ts1Peaks) == 0 || len(ts2Peaks) == 0 {
		return false
	}

	// If ts1 peak occurs before ts2 peak, likely causation
	return ts1Peaks[0].Before(ts2Peaks[0])
}

func (ce *CorrelationEngine) findPeaks(points []DataPoint) []time.Time {
	peaks := []time.Time{}
	
	for i := 1; i < len(points)-1; i++ {
		if points[i].Value > points[i-1].Value && points[i].Value > points[i+1].Value {
			peaks = append(peaks, points[i].Timestamp)
		}
	}

	return peaks
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ce *CorrelationEngine) estimateAffectedUsers(services []string, errorRate float64) int {
	// Simplified estimation
	baseUsers := len(services) * 100
	return int(float64(baseUsers) * errorRate / 100.0)
}

func (ce *CorrelationEngine) describeBusinessImpact(impact CorrelationImpact) string {
	switch impact.Severity {
	case "critical":
		return "Critical business impact - immediate action required"
	case "high":
		return "High business impact - service degradation affecting users"
	case "medium":
		return "Medium business impact - some users experiencing issues"
	default:
		return "Low business impact - monitoring situation"
	}
}

func (ce *CorrelationEngine) describeImpact(value float64, metricName string) string {
	if metricName == "packet_loss" && value > 5 {
		return "High packet loss causing network issues"
	}
	if metricName == "http_errors" && value > 5 {
		return "Elevated error rate affecting users"
	}
	return "Metric value changed"
}

func getComponent(labels map[string]string) string {
	if component, ok := labels["component"]; ok {
		return component
	}
	if pod, ok := labels["pod"]; ok {
		return pod
	}
	if node, ok := labels["node"]; ok {
		return node
	}
	return "unknown"
}

func (ce *CorrelationEngine) getMetricStatus(ref MetricReference) string {
	if math.Abs(ref.Deviation) > 50 {
		return "critical"
	} else if math.Abs(ref.Deviation) > 20 {
		return "warning"
	}
	return "normal"
}

func (ce *CorrelationEngine) getMetricColor(metricType MetricType) string {
	switch metricType {
	case MetricTypeNetwork:
		return "#3B82F6" // Blue
	case MetricTypeApplication:
		return "#10B981" // Green
	case MetricTypeInfrastructure:
		return "#F59E0B" // Orange
	default:
		return "#6B7280" // Gray
	}
}

func (ce *CorrelationEngine) toChartPoints(dataPoints []DataPoint) []ChartPoint {
	points := make([]ChartPoint, len(dataPoints))
	for i, dp := range dataPoints {
		points[i] = ChartPoint{
			X: dp.Timestamp,
			Y: dp.Value,
		}
	}
	return points
}

// GetCorrelations returns all detected correlations
func (ce *CorrelationEngine) GetCorrelations() []Correlation {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	correlations := make([]Correlation, 0, len(ce.correlations))
	for _, corr := range ce.correlations {
		correlations = append(correlations, *corr)
	}

	return correlations
}

// GetCorrelationByID returns a specific correlation
func (ce *CorrelationEngine) GetCorrelationByID(id string) *Correlation {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if corr, exists := ce.correlations[id]; exists {
		return corr
	}

	return nil
}
