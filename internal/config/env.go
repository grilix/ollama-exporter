package config

import (
	"fmt"
	"os"
	"time"
)

// EnvConfig holds environment variable parsing utilities
type EnvConfig struct{}

// NewEnvConfig creates a new environment configuration parser
func NewEnvConfig() *EnvConfig {
	return &EnvConfig{}
}

// GetString returns a string value from environment
func (e *EnvConfig) GetString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetDuration returns a duration value from environment
func (e *EnvConfig) GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsedDuration, err := time.ParseDuration(value); err == nil {
			return parsedDuration
		}
		fmt.Printf("Warning: Invalid %s value '%s', using default '%s'.\n", key, value, defaultValue)
	}
	return defaultValue
}

// GetBool returns a boolean value from environment
func (e *EnvConfig) GetBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if value == "true" || value == "1" || value == "yes" {
			return true
		}
		if value == "false" || value == "0" || value == "no" {
			return false
		}
		fmt.Printf("Warning: Invalid %s value '%s', using default '%t'.\n", key, value, defaultValue)
	}
	return defaultValue
}

// GetInt returns an integer value from environment
func (e *EnvConfig) GetInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", new(int)); err == nil && intValue == 1 {
			var result int
			fmt.Sscanf(value, "%d", &result)
			return result
		}
		fmt.Printf("Warning: Invalid %s value '%s', using default '%d'.\n", key, value, defaultValue)
	}
	return defaultValue
}
