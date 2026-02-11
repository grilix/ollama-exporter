package proxy

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// Forwarder handles request forwarding to Ollama server
type Forwarder struct {
	ollamaURLBase string
	timeout       time.Duration
}

// NewForwarder creates a new forwarder instance
func NewForwarder(ollamaURLBase string, timeout time.Duration) *Forwarder {
	return &Forwarder{
		ollamaURLBase: ollamaURLBase,
		timeout:       timeout,
	}
}

// ForwardRequest forwards a request to the Ollama server and returns the response
func (f *Forwarder) ForwardRequest(c *gin.Context, body []byte) (*http.Response, time.Time, error) {
	ollamaURL, err := url.Parse(f.ollamaURLBase + c.Request.URL.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Ollama URL"})
		return nil, time.Now(), fmt.Errorf("Error parsing Ollama URL: %v", err)
	}

	req, err := http.NewRequest(c.Request.Method, ollamaURL.String(), bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error creating request to Ollama server: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return nil, time.Now(), fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range c.Request.Header {
		if key == "Host" {
			continue
		}

		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: f.timeout}

	responseStart := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach Ollama server"})
		return nil, responseStart, fmt.Errorf("Error forwarding request to Ollama server: %v", err)
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)

	return resp, responseStart, err
}
