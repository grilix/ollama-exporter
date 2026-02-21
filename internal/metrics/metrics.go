package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Version information
	ExporterVersionGauge *prometheus.GaugeVec

	// Tool calls
	OllamaToolCallsTotal *prometheus.CounterVec

	// Request/Response counters
	OllamaTransparentRequestsTotal *prometheus.CounterVec
	OllamaRequestsTotal            *prometheus.CounterVec
	OllamaResponsesTotal           *prometheus.CounterVec

	// Duration histograms
	OllamaResponseSeconds           *prometheus.HistogramVec
	OllamaLoadDurationSeconds       *prometheus.HistogramVec
	OllamaPromptEvalDurationSeconds *prometheus.HistogramVec
	OllamaEvalDurationSeconds       *prometheus.HistogramVec
	OllamaTokensPerSecond           *prometheus.HistogramVec
	OllamaTotalDuration             *prometheus.HistogramVec

	// Token counters
	OllamaTokensProcessedTotal *prometheus.CounterVec
	OllamaTokensGeneratedTotal *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance with all Prometheus metrics initialized
func NewMetrics() *Metrics {
	return &Metrics{
		ExporterVersionGauge: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ollama_exporter_version",
			Help: "Exporter version",
		}, []string{"version"}),

		OllamaToolCallsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ollama_tool_calls_total",
			Help: "Total tool calls",
		}, []string{"model", "api", "type", "tool_type", "tool_name"}),

		OllamaTransparentRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ollama_transparent_requests_total",
			Help: "Total requests passed through",
		}, []string{"method", "endpoint"}),

		OllamaRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ollama_requests_total",
			Help: "Total chat requests",
		}, []string{"model", "api", "type"}),

		OllamaResponsesTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ollama_responses_total",
			Help: "Total responses, with status label",
		}, []string{"model", "api", "type", "status"}),

		OllamaResponseSeconds: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ollama_response_seconds",
			Help:    "Total time spent for the response",
			Buckets: []float64{2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048},
		}, []string{"model", "api", "type"}),

		OllamaLoadDurationSeconds: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "ollama_load_duration_seconds",
			Help: "Time spent loading the model",
		}, []string{"model"}),

		OllamaPromptEvalDurationSeconds: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "ollama_prompt_eval_duration_seconds",
			Help: "Time spent evaluating prompt",
		}, []string{"model"}),

		OllamaEvalDurationSeconds: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "ollama_eval_duration_seconds",
			Help: "Time spent generating the response",
		}, []string{"model"}),

		OllamaTokensProcessedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ollama_tokens_processed_total",
			Help: "Number of tokens processed",
		}, []string{"model", "api", "type"}),

		OllamaTokensGeneratedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ollama_tokens_generated_total",
			Help: "Number of tokens generated",
		}, []string{"model", "api", "type"}),

		OllamaTokensPerSecond: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ollama_tokens_per_second",
			Help:    "Tokens generated per second",
			Buckets: []float64{5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		}, []string{"model"}),

		OllamaTotalDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ollama_total_duration_seconds",
			Help:    "Total duration of the request",
			Buckets: []float64{2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048},
		}, []string{"model"}),
	}
}

type ModelMetrics struct { // TODO: rename, this groups [model, api, type]
	Model       string
	API         string
	RequestType string
}

type ToolUsage struct {
	ModelMetrics *ModelMetrics
	Type         string
	Name         string
}

// OllamaMetrics is a minimal representation of the data needed for metric collection.
type OllamaMetrics struct {
	ModelMetrics *ModelMetrics

	TotalDuration      int64
	LoadDuration       int64
	PromptEvalDuration int64
	PromptEvalCount    int64
	EvalDuration       int64
	EvalCount          int64
}

type OpenAIMetrics struct {
	ModelMetrics *ModelMetrics

	PromptTokens     int64 // processed, TODO?
	CompletionTokens int64 // generated, TODO?
}

// SetVersion sets the exporter version metric
func (m *Metrics) SetVersion(version string) {
	m.ExporterVersionGauge.WithLabelValues(version).Set(1)
}
