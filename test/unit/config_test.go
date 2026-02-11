package config_test

import (
	"testing"
	"time"

	"ollama-exporter/internal/config"
)

func TestNewConfig(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("EXPORTER_PORT", "")
	t.Setenv("OLLAMA_TIMEOUT", "")

	env := config.NewEnvConfig()
	cfg := config.NewConfig(env)

	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
	}

	if cfg.ExporterPort != "8000" {
		t.Errorf("Expected default ExporterPort to be '8000', got '%s'", cfg.ExporterPort)
	}

	if cfg.OllamaHost != "localhost:11434" {
		t.Errorf("Expected default OllamaHost to be 'localhost:11434', got '%s'", cfg.OllamaHost)
	}

	if cfg.OllamaTimeout != 50*time.Minute {
		t.Errorf("Expected default OllamaTimeout to be 50 minutes, got %v", cfg.OllamaTimeout)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     &config.Config{ExporterPort: "8000", OllamaHost: "localhost:11434", OllamaTimeout: 5 * time.Minute},
			wantErr: false,
		},
		{
			name:    "empty port",
			cfg:     &config.Config{ExporterPort: "", OllamaHost: "localhost:11434", OllamaTimeout: 5 * time.Minute},
			wantErr: true,
		},
		{
			name:    "empty host",
			cfg:     &config.Config{ExporterPort: "8000", OllamaHost: "", OllamaTimeout: 5 * time.Minute},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			cfg:     &config.Config{ExporterPort: "8000", OllamaHost: "localhost:11434", OllamaTimeout: -1 * time.Minute},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigInitialize(t *testing.T) {
	cfg := &config.Config{
		OllamaHost: "localhost:11434",
	}

	cfg.Initialize()

	expectedURL := "http://localhost:11434"
	if cfg.OllamaURLBase != expectedURL {
		t.Errorf("Expected OllamaURLBase to be '%s', got '%s'", expectedURL, cfg.OllamaURLBase)
	}
}

func TestConfigGetExporterAddr(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{
			name:     "empty addr",
			cfg:      &config.Config{ExporterAddr: "", ExporterPort: "8000"},
			expected: ":8000",
		},
		{
			name:     "with addr",
			cfg:      &config.Config{ExporterAddr: "127.0.0.1", ExporterPort: "8000"},
			expected: "127.0.0.1:8000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.GetExporterAddr()
			if result != tt.expected {
				t.Errorf("Config.GetExporterAddr() = %v, want %v", result, tt.expected)
			}
		})
	}
}
