package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// Command represents a CLI command
type Command string

const (
	CmdVisualize Command = "visualize"
	CmdHealth    Command = "health"
	CmdIssues    Command = "issues"
	CmdProbe     Command = "probe"
	CmdExport    Command = "export"
	CmdSimulate  Command = "simulate"
)

// Config holds CLI configuration
type Config struct {
	ServerURL string
	Namespace string
	Output    string
	Severity  string
	Format    string
}

// NetworkTopology represents the network topology
type NetworkTopology struct {
	Nodes     []map[string]interface{} `json:"nodes"`
	Edges     []map[string]interface{} `json:"edges"`
	Timestamp string                   `json:"timestamp"`
}

// NetworkIssue represents a detected issue
type NetworkIssue struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Severity         string   `json:"severity"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	AffectedResources []string `json:"affected_resources"`
	Suggestions      []string `json:"suggestions"`
	Timestamp        string   `json:"timestamp"`
}

// ProbeResult represents a connectivity probe result
type ProbeResult struct {
	SourcePod      string `json:"source_pod"`
	SourceNS       string `json:"source_namespace"`
	TargetPod      string `json:"target_pod,omitempty"`
	TargetNS       string `json:"target_namespace,omitempty"`
	TargetSvc      string `json:"target_service,omitempty"`
	TargetIP       string `json:"target_ip"`
	TargetPort     int32  `json:"target_port"`
	Success        bool   `json:"success"`
	LatencyMs      int64  `json:"latency_ms"`
	Error          string `json:"error,omitempty"`
	Timestamp      string `json:"timestamp"`
}

func main() {
	var config Config

	// Global flags
	flag.StringVar(&config.ServerURL, "server", "http://localhost:8080", "Network visualizer server URL")
	flag.StringVar(&config.Output, "output", "", "Output file (default: stdout)")
	flag.StringVar(&config.Format, "format", "table", "Output format: table, json, yaml")

	// Parse command
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := Command(args[0])
	
	// Parse command-specific flags
	cmdArgs := args[1:]
	switch command {
	case CmdVisualize:
		handleVisualize(config, cmdArgs)
	case CmdHealth:
		handleHealth(config, cmdArgs)
	case CmdIssues:
		handleIssues(config, cmdArgs)
	case CmdProbe:
		handleProbe(config, cmdArgs)
	case CmdExport:
		handleExport(config, cmdArgs)
	case CmdSimulate:
		handleSimulate(config, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Kubernetes Network Visualizer CLI

Usage:
  k8s-netvis [flags] <command> [args]

Commands:
  visualize    Display current network topology
  health       Run health checks on the cluster
  issues       List detected network issues
  probe        Run connectivity probes
  export       Export topology data
  simulate     Simulate network policy changes

Global Flags:
  -server string    Network visualizer server URL (default: http://localhost:8080)
  -output string    Output file (default: stdout)
  -format string    Output format: table, json, yaml (default: table)

Examples:
  k8s-netvis visualize --namespace default
  k8s-netvis health --all-namespaces
  k8s-netvis issues --severity critical
  k8s-netvis export --format json --output topology.json
  k8s-netvis simulate --policy new-policy.yaml`)
}

func handleVisualize(config Config, args []string) {
	fs := flag.NewFlagSet("visualize", flag.ExitOnError)
	namespace := fs.String("namespace", "", "Namespace to visualize (empty for all)")
	fs.Parse(args)

	// Fetch topology from server
	url := fmt.Sprintf("%s/api/topology", config.ServerURL)
	if *namespace != "" {
		url += fmt.Sprintf("?namespace=%s", *namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching topology: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var topology NetworkTopology
	if err := json.NewDecoder(resp.Body).Decode(&topology); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		os.Exit(1)
	}

	// Output based on format
	switch config.Format {
	case "json":
		outputJSON(topology, config.Output)
	default:
		printTopologyTable(topology)
	}
}

func handleHealth(config Config, args []string) {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	allNamespaces := fs.Bool("all-namespaces", false, "Check all namespaces")
	namespace := fs.String("namespace", "default", "Namespace to check")
	fs.Parse(args)

	// Fetch health status
	url := fmt.Sprintf("%s/api/health", config.ServerURL)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching health: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Health Status: %s\n", body)

	// Fetch probe results
	url = fmt.Sprintf("%s/api/probes", config.ServerURL)
	resp, err = http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching probes: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var probes []ProbeResult
	if err := json.NewDecoder(resp.Body).Decode(&probes); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding probes: %v\n", err)
		os.Exit(1)
	}

	// Filter by namespace if needed
	if !*allNamespaces {
		filtered := []ProbeResult{}
		for _, probe := range probes {
			if probe.SourceNS == *namespace || probe.TargetNS == *namespace {
				filtered = append(filtered, probe)
			}
		}
		probes = filtered
	}

	printProbesTable(probes)
}

func handleIssues(config Config, args []string) {
	fs := flag.NewFlagSet("issues", flag.ExitOnError)
	severity := fs.String("severity", "", "Filter by severity: critical, high, medium, low")
	issueType := fs.String("type", "", "Filter by type: connectivity, latency, policy, dns, configuration")
	fs.Parse(args)

	// Fetch issues from server
	url := fmt.Sprintf("%s/api/issues", config.ServerURL)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var issues []NetworkIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		os.Exit(1)
	}

	// Filter issues
	filtered := []NetworkIssue{}
	for _, issue := range issues {
		if *severity != "" && issue.Severity != *severity {
			continue
		}
		if *issueType != "" && issue.Type != *issueType {
			continue
		}
		filtered = append(filtered, issue)
	}

	// Output based on format
	switch config.Format {
	case "json":
		outputJSON(filtered, config.Output)
	default:
		printIssuesTable(filtered)
	}
}

func handleProbe(config Config, args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	source := fs.String("source", "", "Source pod (namespace/name)")
	target := fs.String("target", "", "Target pod or service (namespace/name)")
	fs.Parse(args)

	if *source == "" || *target == "" {
		fmt.Fprintf(os.Stderr, "Both source and target are required\n")
		os.Exit(1)
	}

	fmt.Printf("Running probe from %s to %s...\n", *source, *target)

	// Fetch latest probe results
	url := fmt.Sprintf("%s/api/probes", config.ServerURL)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching probes: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var probes []ProbeResult
	if err := json.NewDecoder(resp.Body).Decode(&probes); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding probes: %v\n", err)
		os.Exit(1)
	}

	// Find matching probes
	sourceParts := strings.Split(*source, "/")
	targetParts := strings.Split(*target, "/")

	for _, probe := range probes {
		if len(sourceParts) == 2 && probe.SourceNS == sourceParts[0] && probe.SourcePod == sourceParts[1] {
			if len(targetParts) == 2 {
				if (probe.TargetNS == targetParts[0] && probe.TargetPod == targetParts[1]) ||
					(probe.TargetNS == targetParts[0] && probe.TargetSvc == targetParts[1]) {
					printProbeResult(probe)
				}
			}
		}
	}
}

func handleExport(config Config, args []string) {
	// Fetch topology
	url := fmt.Sprintf("%s/api/topology", config.ServerURL)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching topology: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var topology NetworkTopology
	if err := json.NewDecoder(resp.Body).Decode(&topology); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		os.Exit(1)
	}

	outputJSON(topology, config.Output)
	fmt.Printf("Topology exported successfully\n")
}

func handleSimulate(config Config, args []string) {
	fs := flag.NewFlagSet("simulate", flag.ExitOnError)
	policyFile := fs.String("policy", "", "Path to NetworkPolicy YAML file")
	fs.Parse(args)

	if *policyFile == "" {
		fmt.Fprintf(os.Stderr, "Policy file is required\n")
		os.Exit(1)
	}

	// Read policy file
	policyData, err := os.ReadFile(*policyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading policy file: %v\n", err)
		os.Exit(1)
	}

	// Send to server for simulation
	url := fmt.Sprintf("%s/api/simulate", config.ServerURL)
	resp, err := http.Post(url, "application/yaml", strings.NewReader(string(policyData)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error simulating policy: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Simulation Result:\n%s\n", body)
}

// Helper functions for output formatting

func printTopologyTable(topology NetworkTopology) {
	fmt.Printf("\n=== Network Topology ===\n")
	fmt.Printf("Timestamp: %s\n", topology.Timestamp)
	fmt.Printf("Nodes: %d, Edges: %d\n\n", len(topology.Nodes), len(topology.Edges))

	// Print nodes
	fmt.Println("NODES:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tNAME\tNAMESPACE\tHEALTH\tIP")
	
	for _, node := range topology.Nodes {
		nodeType := getStringField(node, "type")
		name := getStringField(node, "name")
		namespace := getStringField(node, "namespace")
		health := getStringField(node, "health")
		ip := getStringField(node, "pod_ip")
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", nodeType, name, namespace, health, ip)
	}
	w.Flush()

	// Print edges summary
	fmt.Println("\nCONNECTIONS:")
	healthCounts := make(map[string]int)
	for _, edge := range topology.Edges {
		health := getStringField(edge, "health")
		healthCounts[health]++
	}
	
	for health, count := range healthCounts {
		fmt.Printf("  %s: %d\n", health, count)
	}
}

func printIssuesTable(issues []NetworkIssue) {
	if len(issues) == 0 {
		fmt.Println("No issues detected")
		return
	}

	fmt.Printf("\n=== Network Issues (%d) ===\n\n", len(issues))
	
	for i, issue := range issues {
		fmt.Printf("[%s] %s\n", strings.ToUpper(issue.Severity), issue.Title)
		fmt.Printf("  Type: %s\n", issue.Type)
		fmt.Printf("  Description: %s\n", issue.Description)
		
		if len(issue.AffectedResources) > 0 {
			fmt.Printf("  Affected: %s\n", strings.Join(issue.AffectedResources, ", "))
		}
		
		if len(issue.Suggestions) > 0 {
			fmt.Printf("  Suggestions:\n")
			for _, suggestion := range issue.Suggestions {
				fmt.Printf("    - %s\n", suggestion)
			}
		}
		
		if i < len(issues)-1 {
			fmt.Println()
		}
	}
}

func printProbesTable(probes []ProbeResult) {
	fmt.Printf("\n=== Connectivity Probes ===\n\n")
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tTARGET\tSUCCESS\tLATENCY\tERROR")
	
	for _, probe := range probes {
		source := fmt.Sprintf("%s/%s", probe.SourceNS, probe.SourcePod)
		target := ""
		if probe.TargetSvc != "" {
			target = fmt.Sprintf("%s/%s", probe.TargetNS, probe.TargetSvc)
		} else if probe.TargetPod != "" {
			target = fmt.Sprintf("%s/%s", probe.TargetNS, probe.TargetPod)
		} else {
			target = fmt.Sprintf("%s:%d", probe.TargetIP, probe.TargetPort)
		}
		
		success := "✓"
		if !probe.Success {
			success = "✗"
		}
		
		latency := fmt.Sprintf("%dms", probe.LatencyMs)
		if probe.LatencyMs == 0 {
			latency = "-"
		}
		
		error := probe.Error
		if error == "" {
			error = "-"
		} else if len(error) > 50 {
			error = error[:47] + "..."
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", source, target, success, latency, error)
	}
	w.Flush()
}

func printProbeResult(probe ProbeResult) {
	fmt.Printf("\nProbe Result:\n")
	fmt.Printf("  Source: %s/%s\n", probe.SourceNS, probe.SourcePod)
	
	if probe.TargetSvc != "" {
		fmt.Printf("  Target Service: %s/%s\n", probe.TargetNS, probe.TargetSvc)
	} else if probe.TargetPod != "" {
		fmt.Printf("  Target Pod: %s/%s\n", probe.TargetNS, probe.TargetPod)
	}
	
	fmt.Printf("  Target IP: %s:%d\n", probe.TargetIP, probe.TargetPort)
	
	if probe.Success {
		fmt.Printf("  Status: SUCCESS\n")
		fmt.Printf("  Latency: %dms\n", probe.LatencyMs)
	} else {
		fmt.Printf("  Status: FAILED\n")
		fmt.Printf("  Error: %s\n", probe.Error)
	}
	
	fmt.Printf("  Timestamp: %s\n", probe.Timestamp)
}

func outputJSON(data interface{}, outputFile string) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	
	if outputFile != "" {
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		encoder = json.NewEncoder(file)
		encoder.SetIndent("", "  ")
	}
	
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func getStringField(m map[string]interface{}, field string) string {
	if v, ok := m[field]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}