package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"ollama-exporter/internal/metrics"
)

func TestNewCollector(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	if c == nil {
		t.Fatal("NewCollector() returned nil")
	}
}

func TestRecordOpenAIToolsUsage(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test with valid ToolUsage
	usage := metrics.ToolUsage{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		Type: "test-tool-type",
		Name: "test-tool-name",
	}
	c.RecordOpenAIToolsUsage(usage)

	// Verify that the counter was incremented
	counterValue := testutil.ToFloat64(m.OllamaToolCallsTotal.WithLabelValues("test-model", "test-api", "test-type", "test-tool-type", "test-tool-name"))
	if counterValue != 1 {
		t.Errorf("Expected counter value to be 1, got %f", counterValue)
	}

	// Test with nil ModelMetrics (should use "(n/a)" placeholders)
	usageNil := metrics.ToolUsage{
		ModelMetrics: nil,
		Type:         "test-tool-type",
		Name:         "test-tool-name",
	}
	c.RecordOpenAIToolsUsage(usageNil)

	// Verify that the counter was incremented with "(n/a)" values
	counterValueNil := testutil.ToFloat64(m.OllamaToolCallsTotal.WithLabelValues("(n/a)", "(n/a)", "(n/a)", "test-tool-type", "test-tool-name"))
	if counterValueNil != 1 {
		t.Errorf("Expected counter value with nil model to be 1, got %f", counterValueNil)
	}
}

func TestExtractOllamaMetrics(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test with nil OllamaMetrics - should not panic
	c.ExtractOllamaMetrics(nil)

	// Test with zero values - should not panic
	c.ExtractOllamaMetrics(&metrics.OllamaMetrics{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		TotalDuration:      0,
		LoadDuration:       0,
		PromptEvalDuration: 0,
		PromptEvalCount:    0,
		EvalDuration:       0,
		EvalCount:          0,
	})

	// Test with partial values - only TotalDuration > 0
	c.ExtractOllamaMetrics(&metrics.OllamaMetrics{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		TotalDuration:      1000000000, // 1 second
		LoadDuration:       0,
		PromptEvalDuration: 0,
		PromptEvalCount:    0,
		EvalDuration:       0,
		EvalCount:          0,
	})

	// Test with full values
	c.ExtractOllamaMetrics(&metrics.OllamaMetrics{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		TotalDuration:      2000000000, // 2 seconds
		LoadDuration:       1000000000, // 1 second
		PromptEvalDuration: 500000000,  // 0.5 second
		PromptEvalCount:    100,
		EvalDuration:       1500000000, // 1.5 seconds
		EvalCount:          200,
	})

	// Verify counters were updated correctly
	// Test OllamaTokensProcessedTotal counter (should have added 100)
	processedValue := testutil.ToFloat64(m.OllamaTokensProcessedTotal.WithLabelValues("test-model", "test-api", "test-type"))
	if processedValue != 100 {
		t.Errorf("Expected OllamaTokensProcessedTotal value to be 100, got %f", processedValue)
	}

	// Test OllamaTokensGeneratedTotal counter (should have added 200)
	generatedValue := testutil.ToFloat64(m.OllamaTokensGeneratedTotal.WithLabelValues("test-model", "test-api", "test-type"))
	if generatedValue != 200 {
		t.Errorf("Expected OllamaTokensGeneratedTotal value to be 200, got %f", generatedValue)
	}

	// Test with negative values (should be ignored due to > 0 checks)
	c.ExtractOllamaMetrics(&metrics.OllamaMetrics{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		TotalDuration:      -1000000000,
		LoadDuration:       -500000000,
		PromptEvalDuration: -250000000,
		PromptEvalCount:    -10,
		EvalDuration:       -750000000,
		EvalCount:          -20,
	})

	// Verify values didn't change after negative values
	processedValueAfter := testutil.ToFloat64(m.OllamaTokensProcessedTotal.WithLabelValues("test-model", "test-api", "test-type"))
	if processedValueAfter != 100 {
		t.Errorf("Expected OllamaTokensProcessedTotal value to still be 100 after negative values, got %f", processedValueAfter)
	}
}

func TestGatherOpenAIUsage(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test with zero tokens - should not affect counters
	c.GatherOpenAIUsage(metrics.OpenAIMetrics{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		PromptTokens:     0,
		CompletionTokens: 0,
	})

	// Test with positive tokens
	c.GatherOpenAIUsage(metrics.OpenAIMetrics{
		ModelMetrics: &metrics.ModelMetrics{
			Model:       "test-model",
			API:         "test-api",
			RequestType: "test-type",
		},
		PromptTokens:     1000,
		CompletionTokens: 2000,
	})

	// Verify that tokens were recorded
	processedValue := testutil.ToFloat64(m.OllamaTokensProcessedTotal.WithLabelValues("test-model", "test-api", "test-type"))
	if processedValue != 1000 {
		t.Errorf("Expected OllamaTokensProcessedTotal value to be 1000, got %f", processedValue)
	}

	generatedValue := testutil.ToFloat64(m.OllamaTokensGeneratedTotal.WithLabelValues("test-model", "test-api", "test-type"))
	if generatedValue != 2000 {
		t.Errorf("Expected OllamaTokensGeneratedTotal value to be 2000, got %f", generatedValue)
	}
}

func TestRecordRequest(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test RecordRequest
	model := &metrics.ModelMetrics{
		Model:       "test-model",
		API:         "test-api",
		RequestType: "test-type",
	}
	c.RecordRequest(model)

	// Verify that the counter was incremented
	counterValue := testutil.ToFloat64(m.OllamaRequestsTotal.WithLabelValues("test-model", "test-api", "test-type"))
	if counterValue != 1 {
		t.Errorf("Expected counter value to be 1, got %f", counterValue)
	}
}

func TestRecordResponse(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test RecordResponse
	model := &metrics.ModelMetrics{
		Model:       "test-model",
		API:         "test-api",
		RequestType: "test-type",
	}
	c.RecordResponse(model, "200")

	// Verify that the counter was incremented
	counterValue := testutil.ToFloat64(m.OllamaResponsesTotal.WithLabelValues("test-model", "test-api", "test-type", "200"))
	if counterValue != 1 {
		t.Errorf("Expected counter value to be 1, got %f", counterValue)
	}
}

func TestRecordResponseDuration(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test RecordResponseDuration - just verify it doesn't panic and histogram is callable
	model := &metrics.ModelMetrics{
		Model:       "test-model",
		API:         "test-api",
		RequestType: "test-type",
	}
	duration := 2.5
	c.RecordResponseDuration(model, duration)

	// Verify the histogram can be accessed (just check it's non-nil)
	histogram := m.OllamaResponseSeconds.WithLabelValues("test-model", "test-api", "test-type")
	if histogram == nil {
		t.Error("Expected histogram to be non-nil")
	}
}

func TestRecordTransparentRequest(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	c := metrics.NewCollector(m)

	// Test RecordTransparentRequest
	c.RecordTransparentRequest("GET", "/test-endpoint")

	// Verify that the counter was incremented
	counterValue := testutil.ToFloat64(m.OllamaTransparentRequestsTotal.WithLabelValues("GET", "/test-endpoint"))
	if counterValue != 1 {
		t.Errorf("Expected counter value to be 1, got %f", counterValue)
	}
}
