package config

import "fmt"

// Config holds the application configuration
type Config struct {
	Kubeconfig string
	Namespace  string
	Output     string
	Port       int
	Verbose    bool
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Output != "cli" && c.Output != "json" && c.Output != "web" {
		return fmt.Errorf("invalid output format: %s (must be cli, json, or web)", c.Output)
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Port)
	}

	return nil
}
