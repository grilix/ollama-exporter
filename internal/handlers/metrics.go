package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"ollama-exporter/internal/routes"
)

// MetricsHandler handles Prometheus metrics endpoint
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// MetricsHandler handles GET /metrics endpoint
func (h *MetricsHandler) MetricsHandler(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

// Routes returns the route for the metrics handler
func (h *MetricsHandler) Routes() []routes.Route {
	return []routes.Route{{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Handler: h.MetricsHandler,
	}}
}
