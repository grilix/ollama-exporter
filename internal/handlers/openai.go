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

	model := "unknown"
	if m, ok := requestData["model"]; ok {
		if mStr, ok := m.(string); ok {
			model = mStr
		}
	}

	isStreaming := false
	if stream, ok := requestData["stream"]; ok {
		if streamBool, ok := stream.(bool); ok {
			isStreaming = streamBool
		}
	}

	if isStreaming {
		h.collector.RecordRequest(model, "openai", "stream")
		err = h.handleStreamingResponse(c, body, model)
	} else {
		h.collector.RecordRequest(model, "openai", "non_stream")
		err = h.handleRegularResponse(c, body, model)
	}
	if err != nil {
		log.Printf("Error handling response: %v", err)
	}
}

// handleStreamingResponse handles streaming responses for /v1/chat/completions
func (h *OpenAIHandler) handleStreamingResponse(c *gin.Context, body []byte, model string) error {
	var requestBody map[string]interface{}
	if err := json.Unmarshal(body, &requestBody); err != nil {
		h.collector.RecordResponse(model, "openai", "stream", "failed")
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
		h.collector.RecordResponse(model, "openai", "stream", "failed")
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, responseStart, err := h.forwarder.ForwardRequest(c, body)
	if err != nil {
		h.collector.RecordResponse(model, "openai", "stream", "failed")
		return fmt.Errorf("failed to forward request: %w", err)
	}

	defer resp.Body.Close()

	var finalUsage *openai.CompletionUsage

	scanner := bufio.NewScanner(resp.Body)
	c.Stream(func(w io.Writer) bool {
		if !scanner.Scan() {
			return false
		}

		line := scanner.Bytes()
		lineStr := strings.TrimSpace(string(line))
		jsonData := strings.TrimPrefix(lineStr, "data:")

		if len(jsonData) > 0 {
			if isFinal, usage := isOpenAIUsageChunk(jsonData); isFinal {
				finalUsage = usage
			}
			h.collector.RecordOpenAIToolsUsage(jsonData, model, "openai", "stream")
			c.SSEvent("", []byte(jsonData))
		}
		return true
	})
	resultError := scanner.Err()

	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	h.collector.RecordResponseDuration(model, "openai", "stream", responseDuration)

	if finalUsage != nil {
		h.collector.GatherOpenAIUsage(model, *finalUsage, "openai", "stream")
	}

	if resultError != nil {
		h.collector.RecordResponse(model, "openai", "stream", "failed")
		return fmt.Errorf("error reading response: %w", resultError)
	}

	h.collector.RecordResponse(model, "openai", "stream", "success")
	return nil
}

// handleRegularResponse handles regular (non-streaming) responses for /v1/chat/completions
func (h *OpenAIHandler) handleRegularResponse(c *gin.Context, body []byte, model string) error {
	resp, responseStart, err := h.forwarder.ForwardRequest(c, body)
	if err != nil {
		h.collector.RecordResponse(model, "openai", "non_stream", "failed")
		return err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	h.collector.RecordResponseDuration(model, "openai", "non_stream", responseDuration)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read response"})
		h.collector.RecordResponse(model, "openai", "non_stream", "failed")
		return fmt.Errorf("Error reading Ollama response: %w", err)
	}

	c.Writer.Write(responseBody)

	if resp.StatusCode == http.StatusOK {
		var openAIResponse openai.ChatCompletion
		if err := json.Unmarshal(responseBody, &openAIResponse); err == nil {
			h.processResponse(model, "openai", "non_stream", resp, responseStart, nil)
		}
		h.collector.RecordResponse(model, "openai", "non_stream", "success")
	} else {
		h.collector.RecordResponse(model, "openai", "non_stream", "failed")
	}

	return nil
}

// processResponse handles common response processing for OpenAI requests
func (h *OpenAIHandler) processResponse(model, api, requestType string, resp *http.Response, responseStart time.Time, response *OllamaResponse) {
	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	h.collector.RecordResponseDuration(model, api, requestType, responseDuration)

	if resp.StatusCode != http.StatusOK {
		h.collector.RecordResponse(model, api, requestType, "failed")
		return
	}

	h.collector.RecordResponse(model, api, requestType, "success")

	if response != nil {
		metricData := &metrics.OllamaMetrics{
			TotalDuration:      response.TotalDuration,
			LoadDuration:       response.LoadDuration,
			PromptEvalDuration: response.PromptEvalDuration,
			PromptEvalCount:    response.PromptEvalCount,
			EvalDuration:       response.EvalDuration,
			EvalCount:          response.EvalCount,
		}
		h.collector.ExtractOllamaMetrics(metricData, model)
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

// isOpenAIUsageChunk checks if the JSON data contains usage information
func isOpenAIUsageChunk(jsonData string) (bool, *openai.CompletionUsage) {
	if !strings.Contains(jsonData, "\"usage\"") {
		return false, nil
	}

	var chunkData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &chunkData); err != nil {
		return false, nil
	}

	if usageData, ok := chunkData["usage"]; ok {
		if usageBytes, err := json.Marshal(usageData); err == nil {
			var usage openai.CompletionUsage
			if err := json.Unmarshal(usageBytes, &usage); err == nil {
				return true, &usage
			}
		}
	}

	return false, nil
}
