package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// HealthChecker handles health check functionality
type HealthChecker struct {
	ollamaURLBase string
	client        *http.Client
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(ollamaURLBase string) *HealthChecker {
	return &HealthChecker{
		ollamaURLBase: ollamaURLBase,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// VerifyOllamaConnection checks if the Ollama server is reachable
func (h *HealthChecker) VerifyOllamaConnection() error {
	resp, err := h.client.Get(h.ollamaURLBase + "/api/version")
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama server at %s: %w", h.ollamaURLBase, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect to Ollama server. Status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading Ollama response: %w", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return fmt.Errorf("error parsing request body: %w", err)
	}

	if version, ok := responseData["version"]; ok {
		log.Printf("Connected to Ollama server version %s at %s", version, h.ollamaURLBase)
		return nil
	}

	return fmt.Errorf("got unexpected response from %s/api/version", h.ollamaURLBase)
}
