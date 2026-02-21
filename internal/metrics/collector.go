package metrics

// Collector handles metric collection logic
type Collector struct {
	metrics *Metrics
}

// NewCollector creates a new metric collector
func NewCollector(metrics *Metrics) *Collector {
	return &Collector{
		metrics: metrics,
	}
}

func (c *Collector) RecordOpenAIToolsUsage(usage ToolUsage) {
	modelMetrics := usage.ModelMetrics
	if modelMetrics == nil {
		modelMetrics = &ModelMetrics{
			Model:       "(n/a)",
			API:         "(n/a)",
			RequestType: "(n/a)",
		}
	}

	c.metrics.OllamaToolCallsTotal.WithLabelValues(modelMetrics.Model, modelMetrics.API, modelMetrics.RequestType, usage.Type, usage.Name).Inc()
}

func durationSeconds(duration int64) float64 {
	return float64(duration) / 1_000_000_000
}

// ExtractOllamaMetrics extracts and records metrics from Ollama metrics data
func (c *Collector) ExtractOllamaMetrics(m *OllamaMetrics) {
	if m == nil {
		return
	}

	modelMetrics := m.ModelMetrics
	if modelMetrics == nil {
		modelMetrics = &ModelMetrics{
			Model:       "(n/a)",
			API:         "(n/a)",
			RequestType: "(n/a)",
		}
	}

	totalDurationSeconds := durationSeconds(m.TotalDuration)
	loadDurationSeconds := durationSeconds(m.LoadDuration)
	promptEvalTimeSeconds := durationSeconds(m.PromptEvalDuration)
	evalDurationSeconds := durationSeconds(m.EvalDuration)

	if m.TotalDuration > 0 {
		c.metrics.OllamaResponseSeconds.WithLabelValues(modelMetrics.Model, modelMetrics.API, modelMetrics.RequestType).Observe(totalDurationSeconds)
		c.metrics.OllamaTotalDuration.WithLabelValues(modelMetrics.Model).Observe(totalDurationSeconds)
	}
	if m.LoadDuration > 0 {
		c.metrics.OllamaLoadDurationSeconds.WithLabelValues(modelMetrics.Model).Observe(loadDurationSeconds)
	}
	if m.PromptEvalDuration > 0 {
		c.metrics.OllamaPromptEvalDurationSeconds.WithLabelValues(modelMetrics.Model).Observe(promptEvalTimeSeconds)
	}
	if m.PromptEvalCount > 0 {
		c.metrics.OllamaTokensProcessedTotal.WithLabelValues(modelMetrics.Model, modelMetrics.API, modelMetrics.RequestType).Add(float64(m.PromptEvalCount))
	}
	if m.EvalDuration > 0 {
		c.metrics.OllamaEvalDurationSeconds.WithLabelValues(modelMetrics.Model).Observe(evalDurationSeconds)
	}
	if m.EvalCount > 0 {
		c.metrics.OllamaTokensGeneratedTotal.WithLabelValues(modelMetrics.Model, modelMetrics.API, modelMetrics.RequestType).Add(float64(m.EvalCount))
	}
	if m.EvalDuration > 0 && m.EvalCount > 0 {
		tps := float64(m.EvalCount) / evalDurationSeconds
		c.metrics.OllamaTokensPerSecond.WithLabelValues(modelMetrics.Model).Observe(tps)
	}
}

// GatherOpenAIUsage gathers metrics from OpenAI usage data
func (c *Collector) GatherOpenAIUsage(usage OpenAIMetrics) {
	modelMetrics := usage.ModelMetrics
	if modelMetrics == nil {
		modelMetrics = &ModelMetrics{
			Model:       "(n/a)",
			API:         "(n/a)",
			RequestType: "(n/a)",
		}
	}
	if usage.PromptTokens > 0 {
		c.metrics.OllamaTokensProcessedTotal.WithLabelValues(modelMetrics.Model, modelMetrics.API, modelMetrics.RequestType).Add(float64(usage.PromptTokens))
	}
	if usage.CompletionTokens > 0 {
		c.metrics.OllamaTokensGeneratedTotal.WithLabelValues(modelMetrics.Model, modelMetrics.API, modelMetrics.RequestType).Add(float64(usage.CompletionTokens))
	}
}

// RecordRequest records a request metric
func (c *Collector) RecordRequest(model *ModelMetrics) {
	c.metrics.OllamaRequestsTotal.WithLabelValues(model.Model, model.API, model.RequestType).Inc()
}

// RecordResponse records a response metric
func (c *Collector) RecordResponse(model *ModelMetrics, status string) {
	c.metrics.OllamaResponsesTotal.WithLabelValues(model.Model, model.API, model.RequestType, status).Inc()
}

// RecordResponseDuration records response duration
func (c *Collector) RecordResponseDuration(model *ModelMetrics, duration float64) {
	c.metrics.OllamaResponseSeconds.WithLabelValues(model.Model, model.API, model.RequestType).Observe(duration)
}

// RecordTransparentRequest records a transparent proxy request
func (c *Collector) RecordTransparentRequest(method, endpoint string) {
	c.metrics.OllamaTransparentRequestsTotal.WithLabelValues(method, endpoint).Inc()
}
