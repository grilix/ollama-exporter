package integration

import (
	"testing"

	"ollama-exporter/internal/config"
)

func TestConfigIntegration(t *testing.T) {
	env := config.NewEnvConfig()
	cfg := config.NewConfig(env)

	if cfg == nil {
		t.Fatal("Failed to create configuration")
	}

	cfg.Initialize()

	if cfg.OllamaURLBase == "" {
		t.Error("OllamaURLBase should not be empty after initialization")
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Configuration validation failed: %v", err)
	}

	addr := cfg.GetExporterAddr()
	if addr == "" {
		t.Error("Exporter address should not be empty")
	}
}

func TestConfigWithEnvironment(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "test-host:11434")
	t.Setenv("EXPORTER_PORT", "9000")

	env := config.NewEnvConfig()
	cfg := config.NewConfig(env)
	cfg.Initialize()

	expectedURL := "http://test-host:11434"
	if cfg.OllamaURLBase != expectedURL {
		t.Errorf("Expected OllamaURLBase to be '%s', got '%s'", expectedURL, cfg.OllamaURLBase)
	}

	if cfg.ExporterPort != "9000" {
		t.Errorf("Expected ExporterPort to be '9000', got '%s'", cfg.ExporterPort)
	}
}
