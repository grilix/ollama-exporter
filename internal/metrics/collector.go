package metrics

import (
	"strings"
	"encoding/json"

	"github.com/openai/openai-go/v3"
)

type OpenAIToolCallsChunk struct {
	Choices []struct {
		Delta struct {
			ToolCalls []struct {
				Type string `json:"type"`
				Function *struct {
					Name string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function,omitempty"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

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

func (c *Collector) RecordOpenAIToolsUsage(jsonData, model, api, requestType string) {
	// data: {
	//   "id":"chatcmpl-220","object":"chat.completion.chunk","created":1771304907,"model":"ministral-3:3b","system_fingerprint":"fp_ollama","choices":[
	//     {"index":0,"delta":{
	//       "role":"assistant","content":"","tool_calls":[
	//         {"id":"call_7v0ou6qm","index":0,"type":"function","function":{
	//           "name":"bash","arguments":"{\"command\":\"ls --color\",\"description\":\"Lists files in current directory\"}"
	//         }}
	//       ]
	//     },"finish_reason":null}
	//   ]
	// }
	// FIXME: create a type for this?
	if !strings.Contains(jsonData, "\"tool_calls\"") {
		return
	}
	var usage OpenAIToolCallsChunk
	// FIXME: we need to actually check if this is what we expect
	if err := json.Unmarshal([]byte(jsonData), &usage); err != nil {
		return
	}

	for _, choice := range usage.Choices {
		for _, call := range choice.Delta.ToolCalls {
			switch call.Type {
			case "function":
				c.metrics.OllamaToolCallsTotal.WithLabelValues(model, api, requestType, call.Type, call.Function.Name).Inc()
				break
			default:
				c.metrics.OllamaToolCallsTotal.WithLabelValues(model, api, requestType, call.Type, "(N/A)").Inc()
			}
		}
	}

	return
}

// ExtractOllamaMetrics extracts and records metrics from Ollama metrics data
func (c *Collector) ExtractOllamaMetrics(m *OllamaMetrics, model string) {
	if m == nil {
		return
	}

	totalDurationSeconds := float64(m.TotalDuration) / 1_000_000_000
	loadDurationSeconds := float64(m.LoadDuration) / 1_000_000_000
	promptEvalTimeSeconds := float64(m.PromptEvalDuration) / 1_000_000_000
	evalDurationSeconds := float64(m.EvalDuration) / 1_000_000_000

	if m.TotalDuration > 0 {
		c.metrics.OllamaResponseSeconds.WithLabelValues(model, "ollama", "non_stream").Observe(totalDurationSeconds)
		c.metrics.OllamaTotalDuration.WithLabelValues(model).Observe(totalDurationSeconds)
	}
	if m.LoadDuration > 0 {
		c.metrics.OllamaLoadDurationSeconds.WithLabelValues(model).Observe(loadDurationSeconds)
	}
	if m.PromptEvalDuration > 0 {
		c.metrics.OllamaPromptEvalDurationSeconds.WithLabelValues(model).Observe(promptEvalTimeSeconds)
	}
	if m.PromptEvalCount > 0 {
		c.metrics.OllamaTokensProcessedTotal.WithLabelValues(model, "ollama", "non_stream").Add(float64(m.PromptEvalCount))
	}
	if m.EvalDuration > 0 {
		c.metrics.OllamaEvalDurationSeconds.WithLabelValues(model).Observe(evalDurationSeconds)
	}
	if m.EvalCount > 0 {
		c.metrics.OllamaTokensGeneratedTotal.WithLabelValues(model, "ollama", "non_stream").Add(float64(m.EvalCount))
	}
	if m.EvalDuration > 0 && m.EvalCount > 0 {
		tps := float64(m.EvalCount) / float64(m.EvalDuration) * 1_000_000_000
		c.metrics.OllamaTokensPerSecond.WithLabelValues(model).Observe(tps)
	}
}

// GatherOpenAIUsage gathers metrics from OpenAI usage data
func (c *Collector) GatherOpenAIUsage(model string, usage openai.CompletionUsage, api, requestType string) {
	if usage.PromptTokens > 0 {
		c.metrics.OllamaTokensProcessedTotal.WithLabelValues(model, api, requestType).Add(float64(usage.PromptTokens))
	}
	if usage.CompletionTokens > 0 {
		c.metrics.OllamaTokensGeneratedTotal.WithLabelValues(model, api, requestType).Add(float64(usage.CompletionTokens))
	}
}

// RecordRequest records a request metric
func (c *Collector) RecordRequest(model, api, requestType string) {
	c.metrics.OllamaRequestsTotal.WithLabelValues(model, api, requestType).Inc()
}

// RecordResponse records a response metric
func (c *Collector) RecordResponse(model, api, requestType, status string) {
	c.metrics.OllamaResponsesTotal.WithLabelValues(model, api, requestType, status).Inc()
}

// RecordResponseDuration records response duration
func (c *Collector) RecordResponseDuration(model, api, requestType string, duration float64) {
	c.metrics.OllamaResponseSeconds.WithLabelValues(model, api, requestType).Observe(duration)
}

// RecordTransparentRequest records a transparent proxy request
func (c *Collector) RecordTransparentRequest(method, endpoint string) {
	c.metrics.OllamaTransparentRequestsTotal.WithLabelValues(method, endpoint).Inc()
}
