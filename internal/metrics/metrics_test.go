package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"ollama-exporter/internal/metrics"
)

func TestNewMetricsInitializesAll(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()

	// Check that all metric fields are non-nil
	if m.ExporterVersionGauge == nil {
		t.Error("ExporterVersionGauge should not be nil")
	}

	if m.OllamaToolCallsTotal == nil {
		t.Error("OllamaToolCallsTotal should not be nil")
	}

	if m.OllamaTransparentRequestsTotal == nil {
		t.Error("OllamaTransparentRequestsTotal should not be nil")
	}

	if m.OllamaRequestsTotal == nil {
		t.Error("OllamaRequestsTotal should not be nil")
	}

	if m.OllamaResponsesTotal == nil {
		t.Error("OllamaResponsesTotal should not be nil")
	}

	if m.OllamaResponseSeconds == nil {
		t.Error("OllamaResponseSeconds should not be nil")
	}

	if m.OllamaLoadDurationSeconds == nil {
		t.Error("OllamaLoadDurationSeconds should not be nil")
	}

	if m.OllamaPromptEvalDurationSeconds == nil {
		t.Error("OllamaPromptEvalDurationSeconds should not be nil")
	}

	if m.OllamaEvalDurationSeconds == nil {
		t.Error("OllamaEvalDurationSeconds should not be nil")
	}

	if m.OllamaTokensPerSecond == nil {
		t.Error("OllamaTokensPerSecond should not be nil")
	}

	if m.OllamaTotalDuration == nil {
		t.Error("OllamaTotalDuration should not be nil")
	}

	if m.OllamaTokensProcessedTotal == nil {
		t.Error("OllamaTokensProcessedTotal should not be nil")
	}

	if m.OllamaTokensGeneratedTotal == nil {
		t.Error("OllamaTokensGeneratedTotal should not be nil")
	}
}

func TestNewMetricsLabels(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()

	// Test ExporterVersionGauge labels
	if _, err := m.ExporterVersionGauge.GetMetricWithLabelValues("test"); err != nil {
		t.Error("ExporterVersionGauge should have a metric with version label", err)
	}

	// Test OllamaToolCallsTotal labels
	if _, err := m.OllamaToolCallsTotal.GetMetricWithLabelValues("test-model", "test-api", "test-type", "test-tool-type", "test-tool-name"); err != nil {
		t.Error("OllamaToolCallsTotal should have a metric with expected labels", err)
	}

	// Test OllamaTransparentRequestsTotal labels
	if _, err := m.OllamaTransparentRequestsTotal.GetMetricWithLabelValues("GET", "/test"); err != nil {
		t.Error("OllamaTransparentRequestsTotal should have a metric with expected labels", err)
	}

	// Test OllamaRequestsTotal labels
	if _, err := m.OllamaRequestsTotal.GetMetricWithLabelValues("test-model", "test-api", "test-type"); err != nil {
		t.Error("OllamaRequestsTotal should have a metric with expected labels", err)
	}

	// Test OllamaResponsesTotal labels
	if _, err := m.OllamaResponsesTotal.GetMetricWithLabelValues("test-model", "test-api", "test-type", "200"); err != nil {
		t.Error("OllamaResponsesTotal should have a metric with expected labels", err)
	}

	// Test OllamaResponseSeconds labels
	if _, err := m.OllamaResponseSeconds.GetMetricWithLabelValues("test-model", "test-api", "test-type"); err != nil {
		t.Error("OllamaResponseSeconds should have a metric with expected labels", err)
	}

	// Test OllamaLoadDurationSeconds labels
	if _, err := m.OllamaLoadDurationSeconds.GetMetricWithLabelValues("test-model"); err != nil {
		t.Error("OllamaLoadDurationSeconds should have a metric with expected labels", err)
	}

	// Test OllamaPromptEvalDurationSeconds labels
	if _, err := m.OllamaPromptEvalDurationSeconds.GetMetricWithLabelValues("test-model"); err != nil {
		t.Error("OllamaPromptEvalDurationSeconds should have a metric with expected labels", err)
	}

	// Test OllamaEvalDurationSeconds labels
	if _, err := m.OllamaEvalDurationSeconds.GetMetricWithLabelValues("test-model"); err != nil {
		t.Error("OllamaEvalDurationSeconds should have a metric with expected labels", err)
	}

	// Test OllamaTokensProcessedTotal labels
	if _, err := m.OllamaTokensProcessedTotal.GetMetricWithLabelValues("test-model", "test-api", "test-type"); err != nil {
		t.Error("OllamaTokensProcessedTotal should have a metric with expected labels", err)
	}

	// Test OllamaTokensGeneratedTotal labels
	if _, err := m.OllamaTokensGeneratedTotal.GetMetricWithLabelValues("test-model", "test-api", "test-type"); err != nil {
		t.Error("OllamaTokensGeneratedTotal should have a metric with expected labels", err)
	}

	// Test OllamaTokensPerSecond labels
	if _, err := m.OllamaTokensPerSecond.GetMetricWithLabelValues("test-model"); err != nil {
		t.Error("OllamaTokensPerSecond should have a metric with expected labels", err)
	}

	// Test OllamaTotalDuration labels
	if _, err := m.OllamaTotalDuration.GetMetricWithLabelValues("test-model"); err != nil {
		t.Error("OllamaTotalDuration should have a metric with expected labels", err)
	}
}

func TestMetricsSetVersion(t *testing.T) {
	originalRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = originalRegistry
	})

	m := metrics.NewMetrics()
	version := "test-version"
	m.SetVersion(version)

	// Verify that the gauge value is set to 1
	gaugeValue := testutil.ToFloat64(m.ExporterVersionGauge.WithLabelValues(version))
	if gaugeValue != 1 {
		t.Errorf("Expected gauge value to be 1, got %f", gaugeValue)
	}
}
