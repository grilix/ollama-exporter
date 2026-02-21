package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ollama-exporter/internal/metrics"
	"ollama-exporter/internal/routes"
)

// OllamaHandler handles Ollama API endpoints
type OllamaHandler struct {
	collector *metrics.Collector
	forwarder Forwarder
}

// Forwarder interface for request forwarding
type Forwarder interface {
	ForwardRequest(c *gin.Context, body []byte) (*http.Response, time.Time, error)
}

// NewOllamaHandler creates a new Ollama handler
func NewOllamaHandler(collector *metrics.Collector, forwarder Forwarder) *OllamaHandler {
	return &OllamaHandler{
		collector: collector,
		forwarder: forwarder,
	}
}

// ChatHandler handles POST requests to /api/chat or /api/generate
func (h *OllamaHandler) ChatHandler(c *gin.Context) {
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
		API:         "ollama",
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

// handleStreamingResponse handles streaming responses
func (h *OllamaHandler) handleStreamingResponse(c *gin.Context, body []byte, modelMetrics *metrics.ModelMetrics) error {
	resp, responseStart, err := h.forwarder.ForwardRequest(c, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to forward request"})
		h.collector.RecordResponse(modelMetrics, "failed")
		return err
	}

	defer resp.Body.Close()

	var finalChunkData *OllamaResponse

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > 0 {
			var chunkData OllamaChunk
			if err := json.Unmarshal(line, &chunkData); err == nil {
				if chunkData.Done {
					finalChunkData = &OllamaResponse{
						TotalDuration:      chunkData.TotalDuration,
						LoadDuration:       chunkData.LoadDuration,
						PromptEvalDuration: chunkData.PromptEvalDuration,
						PromptEvalCount:    chunkData.PromptEvalCount,
						EvalDuration:       chunkData.EvalDuration,
						EvalCount:          chunkData.EvalCount,
						Done:               chunkData.Done,
					}
				}
			}
		}
		line = append(line, '\n')
		c.Writer.Write(line)
		c.Writer.Flush()
	}

	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	h.collector.RecordResponseDuration(modelMetrics, responseDuration)

	if err := scanner.Err(); err != nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("error reading stream: %w", err)
	}

	if finalChunkData == nil {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("final chunk not in response for model: %s", modelMetrics.Model)
	}

	h.processResponse(modelMetrics, resp, finalChunkData)
	return nil
}

// handleRegularResponse handles regular (non-streaming) responses
func (h *OllamaHandler) handleRegularResponse(c *gin.Context, body []byte, modelMetrics *metrics.ModelMetrics) error {
	resp, responseStart, err := h.forwarder.ForwardRequest(c, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to forward request"})
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
		return fmt.Errorf("failed to read Ollama response: %v", err)
	}

	c.Writer.Write(responseBody)

	if resp.StatusCode != http.StatusOK {
		h.collector.RecordResponse(modelMetrics, "failed")
		return fmt.Errorf("response was not success: status=%d", resp.StatusCode)
	}

	var responseData OllamaResponse
	if err := json.Unmarshal(responseBody, &responseData); err == nil {
		h.processResponse(modelMetrics, resp, &responseData)
	}
	return nil
}

// processResponse handles common response processing for Ollama requests
func (h *OllamaHandler) processResponse(modelMetrics *metrics.ModelMetrics, resp *http.Response, response *OllamaResponse) {
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

// Routes returns the routes for the Ollama handler
func (h *OllamaHandler) Routes() []routes.Route {
	return []routes.Route{{
		Method:  "POST",
		Path:    "/api/chat",
		Handler: h.ChatHandler,
	}, {
		Method:  "POST",
		Path:    "/api/generate",
		Handler: h.ChatHandler,
	}}
}
