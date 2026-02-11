package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"

	"ollama-exporter/internal/metrics"
	"ollama-exporter/internal/routes"
)

// Proxy handles proxy functionality for forwarding requests to Ollama
type Proxy struct {
	collector *metrics.Collector
	ollamaURL *url.URL
}

// NewProxy creates a new proxy instance
func NewProxy(collector *metrics.Collector, ollamaURLBase string) (*Proxy, error) {
	ollamaURL, err := url.Parse(ollamaURLBase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Ollama URL: %w", err)
	}

	return &Proxy{
		collector: collector,
		ollamaURL: ollamaURL,
	}, nil
}

// Handler creates a gin handler function that forwards requests to the Ollama server
func (p *Proxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		proxy := httputil.NewSingleHostReverseProxy(p.ollamaURL)
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.URL.Scheme = p.ollamaURL.Scheme
			req.URL.Host = p.ollamaURL.Host
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery
			req.Host = p.ollamaURL.Host
		}

		wrappedWriter := c.Writer
		p.collector.RecordTransparentRequest(c.Request.Method, c.Request.URL.Path)
		proxy.ServeHTTP(wrappedWriter, c.Request)
	}
}

// Routes returns the route for NoRoute handling
func (p *Proxy) Routes() []routes.Route {
	// This is a special case route for NoRoute handling
	return []routes.Route{}
}
