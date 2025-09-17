package config

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid cli config",
			config: Config{
				Output: "cli",
				Port:   8080,
			},
			wantErr: false,
		},
		{
			name: "valid json config",
			config: Config{
				Output: "json",
				Port:   8080,
			},
			wantErr: false,
		},
		{
			name: "valid web config",
			config: Config{
				Output: "web",
				Port:   8080,
			},
			wantErr: false,
		},
		{
			name: "invalid output format",
			config: Config{
				Output: "invalid",
				Port:   8080,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			config: Config{
				Output: "cli",
				Port:   0,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			config: Config{
				Output: "cli",
				Port:   70000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
