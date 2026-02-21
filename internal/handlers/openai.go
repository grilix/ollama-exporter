package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/respjson"

	"ollama-exporter/internal/metrics"
	"ollama-exporter/internal/routes"
)

// OpenAIHandler handles OpenAI API endpoints
type OpenAIHandler struct {
	collector *metrics.Collector
	forwarder Forwarder
}

// NewOpenAIHandler creates a new OpenAI handler
func NewOpenAIHandler(collector *metrics.Collector, forwarder Forwarder) *OpenAIHandler {
	return &OpenAIHandler{
		collector: collector,
		forwarder: forwarder,
	}
}

// CompletionsHandler handles POST /v1/chat/completions endpoint
func (h *OpenAIHandler) CompletionsHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	defer c.Request.Body.Close()

	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		log.Printf("Error parsing request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	modelMetrics := &metrics.ModelMetrics{
		Model:       "unknown",
		API:         "openai",
		RequestType: "non_stream",
	}

	if m, ok := requestData["model"]; ok {
		if mStr, ok := m.(string); ok {
			modelMetrics.Model = mStr
		}
	}

	isStreaming := false
	if stream, ok := requestData["stream"]; ok {
		if streamBool, ok := stream.(bool); ok {
			isStreaming = streamBool
			if streamBool {
				modelMetrics.RequestType = "stream"
			}
		}
	}

	h.collector.RecordRequest(modelMetrics)

	if isStreaming {
		err = h.handleStreamingResponse(c, body, modelMetrics)
	} else {
		err = h.handleRegularResponse(c, body, modelMetrics)
	}
	if err != nil {
		log.Printf("Error handling response: %v", err)
	}
}

// handleStreamingResponse handles streaming responses for /v1/chat/completions
func (h *OpenAIHandler) handleStreamingResponse(c *gin.Context, body []byte, modelMetrics *metrics.ModelMetrics) error {
	var requestBody map[string]interface{}
	if err := json.Unmarshal(body, &requestBody); err != nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("failed to parse request: %w", err)
	}
	if streamOptions, ok := requestBody["stream_options"].(map[string]interface{}); ok {
		streamOptions["include_usage"] = true
		requestBody["stream_options"] = streamOptions
	} else {
		requestBody["stream_options"] = map[string]interface{}{"include_usage": true}
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, responseStart, err := h.forwarder.ForwardRequest(c, body)
	if err != nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("failed to forward request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Upstream error[%d]", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	c.Stream(func(w io.Writer) bool {
		if !scanner.Scan() {
			return false
		}

		line := scanner.Bytes()
		lineStr := strings.TrimSpace(string(line))
		jsonData := strings.TrimPrefix(lineStr, "data:")

		if len(jsonData) > 0 {
			var chunkData openai.ChatCompletionChunk
			c.SSEvent("", []byte(jsonData))

			if err := json.Unmarshal([]byte(jsonData), &chunkData); err == nil {
				extractChunkMetrics(modelMetrics, &chunkData, h.collector)
			}
		}
		return true
	})
	resultError := scanner.Err()

	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	h.collector.RecordResponseDuration(modelMetrics, responseDuration)

	if resultError != nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("error reading response: %w", resultError)
	}

	h.collector.RecordResponse(modelMetrics, "success")
	return nil
}

// handleRegularResponse handles regular (non-streaming) responses for /v1/chat/completions
func (h *OpenAIHandler) handleRegularResponse(c *gin.Context, body []byte, modelMetrics *metrics.ModelMetrics) error {
	resp, responseStart, err := h.forwarder.ForwardRequest(c, body)
	if err != nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	h.collector.RecordResponseDuration(modelMetrics, responseDuration)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read response"})
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("Error reading Ollama response: %w", err)
	}

	c.Writer.Write(responseBody)

	if resp.StatusCode == http.StatusOK {
		var openAIResponse openai.ChatCompletion
		if err := json.Unmarshal(responseBody, &openAIResponse); err == nil {
			h.processResponse(modelMetrics, resp, nil)

			h.collector.GatherOpenAIUsage(metrics.OpenAIMetrics{
				ModelMetrics:     modelMetrics,
				PromptTokens:     openAIResponse.Usage.PromptTokens,
				CompletionTokens: openAIResponse.Usage.CompletionTokens,
			})
		}
		h.collector.RecordResponse(modelMetrics, "success")
	} else {
		h.collector.RecordResponse(modelMetrics, "failed")
	}

	return nil
}

// processResponse handles common response processing for OpenAI requests
func (h *OpenAIHandler) processResponse(modelMetrics *metrics.ModelMetrics, resp *http.Response, response *OllamaResponse) {
	if resp.StatusCode != http.StatusOK {
		h.collector.RecordResponse(modelMetrics, "failed")
		return
	}

	h.collector.RecordResponse(modelMetrics, "success")

	if response != nil {
		metricData := &metrics.OllamaMetrics{
			ModelMetrics:       modelMetrics,
			TotalDuration:      response.TotalDuration,
			LoadDuration:       response.LoadDuration,
			PromptEvalDuration: response.PromptEvalDuration,
			PromptEvalCount:    response.PromptEvalCount,
			EvalDuration:       response.EvalDuration,
			EvalCount:          response.EvalCount,
		}
		h.collector.ExtractOllamaMetrics(metricData)
	}
}

// Routes returns the routes for the OpenAI handler
func (h *OpenAIHandler) Routes() []routes.Route {
	return []routes.Route{{
		Method:  "POST",
		Path:    "/v1/chat/completions",
		Handler: h.CompletionsHandler,
	}}
}

func extractChunkMetrics(modelMetrics *metrics.ModelMetrics, chunkData *openai.ChatCompletionChunk, collector *metrics.Collector) {
	if respjson.Field.Valid(chunkData.JSON.Usage) {
		collector.GatherOpenAIUsage(metrics.OpenAIMetrics{
			ModelMetrics:     modelMetrics,
			PromptTokens:     chunkData.Usage.PromptTokens,
			CompletionTokens: chunkData.Usage.CompletionTokens,
		})
	}
	for _, choice := range chunkData.Choices {
		for _, call := range choice.Delta.ToolCalls {
			usage := metrics.ToolUsage{
				ModelMetrics: modelMetrics,
				Type:         call.Type,
			}
			switch call.Type {
			case "function":
				usage.Name = call.Function.Name
			default:
				usage.Name = "(N/A)" // TODO: should this be ""?
			}
			collector.RecordOpenAIToolsUsage(usage)
		}
	}
}
