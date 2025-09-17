package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/christine33-creator/k8-network-visualizer/internal/cli"
	"github.com/christine33-creator/k8-network-visualizer/internal/config"
)

func main() {
	var (
		kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig file (uses in-cluster config if empty)")
		namespace  = flag.String("namespace", "", "Kubernetes namespace to analyze (empty for all namespaces)")
		output     = flag.String("output", "cli", "Output format: cli, json, web")
		port       = flag.Int("port", 8080, "Port for web interface")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *verbose {
		log.SetOutput(os.Stdout)
	}

	cfg := &config.Config{
		Kubeconfig: *kubeconfig,
		Namespace:  *namespace,
		Output:     *output,
		Port:       *port,
		Verbose:    *verbose,
	}

	if err := cli.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
