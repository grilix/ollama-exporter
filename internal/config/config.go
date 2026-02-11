package config

import (
	"fmt"
	"time"
)

// Config holds all configuration for the ollama-exporter
type Config struct {
	ExporterAddr string
	ExporterPort string

	OllamaHost    string
	OllamaTimeout time.Duration
	OllamaURLBase string

	Version string
}

// NewConfig creates a new configuration instance with default values
func NewConfig(env *EnvConfig) *Config {
	return &Config{
		ExporterAddr:  env.GetString("EXPORTER_ADDR", ""),
		ExporterPort:  env.GetString("EXPORTER_PORT", "8000"),
		OllamaHost:    env.GetString("OLLAMA_HOST", "localhost:11434"),
		OllamaTimeout: env.GetDuration("OLLAMA_TIMEOUT", 50*time.Minute),
		Version:       "dev",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ExporterPort == "" {
		return fmt.Errorf("exporter port cannot be empty")
	}

	if c.OllamaHost == "" {
		return fmt.Errorf("ollama host cannot be empty")
	}

	if c.OllamaTimeout <= 0 {
		return fmt.Errorf("ollama timeout must be positive")
	}

	return nil
}

// Initialize sets up the configuration after creation
func (c *Config) Initialize() {
	c.OllamaURLBase = fmt.Sprintf("http://%s", c.OllamaHost)
}

// GetExporterAddr returns the full exporter address
func (c *Config) GetExporterAddr() string {
	return c.ExporterAddr + ":" + c.ExporterPort
}
