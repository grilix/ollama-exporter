package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/openai/openai-go/v3"
)

var ollamaUrlBase = fmt.Sprintf("http://%s", envValue("OLLAMA_HOST", "localhost:11434"))
var ollamaTimeout = envDurationValue("OLLAMA_TIMEOUT", 50*time.Minute)

var (
	ollamaTransparentRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ollama_transparent_requests_total",
		Help: "Total requests passed through",
	}, []string{"method", "endpoint"})

	ollamaRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ollama_requests_total",
		Help: "Total chat requests",
	}, []string{"model", "api", "type"})

	ollamaResponsesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ollama_responses_total",
		Help: "Total responses, with status label",
	}, []string{"model", "api", "type", "status"})

	ollamaResponseSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ollama_response_seconds",
		Help:    "Total time spent for the response",
		Buckets: []float64{2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048},
	}, []string{"model", "api", "type"})

	ollamaLoadDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ollama_load_duration_seconds",
		Help: "Time spent loading the model",
	}, []string{"model"})

	ollamaPromptEvalDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ollama_prompt_eval_duration_seconds",
		Help: "Time spent evaluating prompt",
	}, []string{"model"})

	ollamaEvalDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "ollama_eval_duration_seconds",
		Help: "Time spent generating the response",
	}, []string{"model"})

	ollamaTokensProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ollama_tokens_processed_total",
		Help: "Number of tokens processed",
	}, []string{"model", "api", "type"})

	ollamaTokensGeneratedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ollama_tokens_generated_total",
		Help: "Number of tokens generated",
	}, []string{"model", "api", "type"})

	ollamaTokensPerSecond = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ollama_tokens_per_second",
		Help:    "Tokens generated per second",
		Buckets: []float64{5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	}, []string{"model"})

	ollamaTotalDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ollama_total_duration_seconds",
		Help:    "Total duration of the request",
		Buckets: []float64{2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048},
	}, []string{"model"})
)

// OllamaResponse represents the structure of Ollama API responses
type OllamaResponse struct {
	TotalDuration      int64 `json:"total_duration"`
	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	PromptEvalCount    int64 `json:"prompt_eval_count"`
	EvalDuration       int64 `json:"eval_duration"`
	EvalCount          int64 `json:"eval_count"`
	Done               bool  `json:"done"`
}

// OllamaChunk represents a stream chunk from Ollama API
type OllamaChunk struct {
	TotalDuration      int64 `json:"total_duration"`
	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	PromptEvalCount    int64 `json:"prompt_eval_count"`
	EvalDuration       int64 `json:"eval_duration"`
	EvalCount          int64 `json:"eval_count"`
	Done               bool  `json:"done"`
}

// Extract and record metrics from Ollama response data
func extractOllamaMetrics(resp *OllamaResponse, model string) {
	if resp == nil {
		return
	}

	totalDurationSeconds := float64(resp.TotalDuration) / 1_000_000_000
	loadDurationSeconds := float64(resp.LoadDuration) / 1_000_000_000
	promptEvalTimeSeconds := float64(resp.PromptEvalDuration) / 1_000_000_000
	evalDurationSeconds := float64(resp.EvalDuration) / 1_000_000_000

	if resp.TotalDuration > 0 {
		ollamaResponseSeconds.WithLabelValues(model, "ollama", "non_stream").Observe(totalDurationSeconds)
		ollamaTotalDuration.WithLabelValues(model).Observe(totalDurationSeconds)
	}
	if resp.LoadDuration > 0 {
		ollamaLoadDurationSeconds.WithLabelValues(model).Observe(loadDurationSeconds)
	}
	if resp.PromptEvalDuration > 0 {
		ollamaPromptEvalDurationSeconds.WithLabelValues(model).Observe(promptEvalTimeSeconds)
	}
	if resp.PromptEvalCount > 0 {
		ollamaTokensProcessedTotal.WithLabelValues(model, "ollama", "non_stream").Add(float64(resp.PromptEvalCount))
	}
	if resp.EvalDuration > 0 {
		ollamaEvalDurationSeconds.WithLabelValues(model).Observe(evalDurationSeconds)
	}
	if resp.EvalCount > 0 {
		ollamaTokensGeneratedTotal.WithLabelValues(model, "ollama", "non_stream").Add(float64(resp.EvalCount))
	}
	if resp.EvalDuration > 0 && resp.EvalCount > 0 {
		tps := float64(resp.EvalCount) / float64(resp.EvalDuration) * 1_000_000_000
		ollamaTokensPerSecond.WithLabelValues(model).Observe(tps)
	}
}

// verifyOllamaConnection checks if the Ollama server is reachable
func verifyOllamaConnection() error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(ollamaUrlBase + "/api/version")
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama server at %s: %w", ollamaUrlBase, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect to Ollama server. Status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading Ollama response: %w", err)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return fmt.Errorf("error parsing request body: %w", err)
	}

	if version, ok := responseData["version"]; ok {
		log.Printf("Connected to Ollama server version %s at %s", version, ollamaUrlBase)
		return nil
	}

	return fmt.Errorf("got unexpected response from %s/api/version", ollamaUrlBase)
}

// makeProxyHandler creates a function that forwards requests to the Ollama server
func makeProxyHandler() (func(*gin.Context), error) {
	ollamaURL, err := url.Parse(ollamaUrlBase)
	if err != nil {
		log.Printf("Error parsing Ollama URL[%s]: %v", ollamaUrlBase, err)
		return func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Ollama URL"})
		}, fmt.Errorf("Error parsing Ollama URL: %w", err)
	}

	return func(c *gin.Context) {
		proxy := httputil.NewSingleHostReverseProxy(ollamaURL)
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.URL.Scheme = ollamaURL.Scheme
			req.URL.Host = ollamaURL.Host
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery
			req.Host = ollamaURL.Host
		}

		wrappedWriter := c.Writer
		ollamaTransparentRequestsTotal.WithLabelValues(c.Request.URL.Path, c.Request.Method).Inc()
		proxy.ServeHTTP(wrappedWriter, c.Request)
	}, nil
}

// ollamaChatHandler handles POST requests to /api/chat or /api/generate
func ollamaChatHandler(c *gin.Context) {
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
		ollamaRequestsTotal.WithLabelValues(model, "ollama", "stream").Inc()
		err = handleOllamaStreamingResponse(c, body, model)
	} else {
		ollamaRequestsTotal.WithLabelValues(model, "ollama", "non_stream").Inc()
		err = handleOllamaRegularResponse(c, body, model)
	}
	if err != nil {
		log.Printf("Error handling response: %v", err)
	}
}

// openAICompletionsHandler handles POST /v1/chat/completions endpoint
func openAICompletionsHandler(c *gin.Context) {
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
		ollamaRequestsTotal.WithLabelValues(model, "openai", "stream").Inc()
		err = handleCompletionsStreamingResponse(c, body, model)
	} else {
		ollamaRequestsTotal.WithLabelValues(model, "openai", "non_stream").Inc()
		err = handleCompletionsRegularResponse(c, body, model)
	}
	if err != nil {
		log.Printf("Error handling response: %v", err)
	}
}

func forwardRequest(c *gin.Context, body []byte) (*http.Response, time.Time, error) {
	ollamaURL, err := url.Parse(ollamaUrlBase + c.Request.URL.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Ollama URL"})
		return nil, time.Now(), fmt.Errorf("Error parsing Ollama URL: %v", err)
	}

	req, err := http.NewRequest(c.Request.Method, ollamaURL.String(), bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return nil, time.Now(), fmt.Errorf("Error creating request: %v", err)
	}

	for key, values := range c.Request.Header {
		if key == "Host" {
			continue
		}

		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: ollamaTimeout}

	responseStart := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach Ollama"})
		return nil, responseStart, fmt.Errorf("Error forwarding request to Ollama: %v", err)
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)

	return resp, responseStart, err
}

// handleCompletionsStreamingResponse handles streaming responses for /v1/chat/completions
func handleCompletionsStreamingResponse(c *gin.Context, body []byte, model string) error {
	var requestBody map[string]interface{}
	if err := json.Unmarshal(body, &requestBody); err != nil {
		ollamaResponsesTotal.WithLabelValues(model, "openai", "stream", "failed").Inc()
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
		ollamaResponsesTotal.WithLabelValues(model, "openai", "stream", "failed").Inc()
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, responseStart, err := forwardRequest(c, body)
	if err != nil {
		ollamaResponsesTotal.WithLabelValues(model, "openai", "stream", "failed").Inc()
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
			c.SSEvent("", []byte(jsonData))
		}
		return true
	})
	resultError := scanner.Err()

	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	ollamaResponseSeconds.WithLabelValues(model, "openai", "stream").Observe(responseDuration)

	if finalUsage != nil {
		gatherOpenAIUsage(model, *finalUsage, "openai", "stream")
	}

	if resultError != nil {
		ollamaResponsesTotal.WithLabelValues(model, "openai", "stream", "failed").Inc()
		return fmt.Errorf("error reading response: %w", resultError)
	}

	ollamaResponsesTotal.WithLabelValues(model, "openai", "stream", "success").Inc()
	return nil
}

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

func gatherOpenAIUsage(model string, usage openai.CompletionUsage, api, requestType string) {
	if usage.PromptTokens > 0 {
		ollamaTokensProcessedTotal.WithLabelValues(model, api, requestType).Add(float64(usage.PromptTokens))
	}
	if usage.CompletionTokens > 0 {
		ollamaTokensGeneratedTotal.WithLabelValues(model, api, requestType).Add(float64(usage.CompletionTokens))
	}
}

// handleCompletionsRegularResponse handles regular (non-streaming) responses for /v1/chat/completions
func handleCompletionsRegularResponse(c *gin.Context, body []byte, model string) error {
	resp, responseStart, err := forwardRequest(c, body)
	if err != nil {
		ollamaResponsesTotal.WithLabelValues(model, "openai", "non_stream", "failed").Inc()
		return err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	ollamaResponseSeconds.WithLabelValues(model, "openai", "non_stream").Observe(responseDuration)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read response"})
		ollamaResponsesTotal.WithLabelValues(model, "openai", "non_stream", "failed").Inc()
		return fmt.Errorf("Error reading Ollama response: %w", err)
	}

	c.Writer.Write(responseBody)

	if resp.StatusCode == http.StatusOK {
		var openAIResponse openai.ChatCompletion
		if err := json.Unmarshal(responseBody, &openAIResponse); err == nil {
			gatherOpenAIUsage(model, openAIResponse.Usage, "openai", "non_stream")
		}
		ollamaResponsesTotal.WithLabelValues(model, "openai", "non_stream", "success").Inc()
	} else {
		ollamaResponsesTotal.WithLabelValues(model, "openai", "non_stream", "failed").Inc()
	}

	return nil
}

// handleOllamaStreamingResponse handles streaming responses
func handleOllamaStreamingResponse(c *gin.Context, body []byte, model string) error {
	resp, responseStart, err := forwardRequest(c, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to forward request"})
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "stream", "failed").Inc()
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
	ollamaResponseSeconds.WithLabelValues(model, "ollama", "stream").Observe(responseDuration)

	if err := scanner.Err(); err != nil {
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "stream", "failed").Inc()
		return fmt.Errorf("error reading stream: %w", err)
	}

	if finalChunkData == nil {
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "stream", "failed").Inc()
		return fmt.Errorf("final chunk not in response for model: %s", model)
	}

	ollamaResponsesTotal.WithLabelValues(model, "ollama", "stream", "success").Inc()
	extractOllamaMetrics(finalChunkData, model)
	return nil
}

// handleOllamaRegularResponse handles regular (non-streaming) responses
func handleOllamaRegularResponse(c *gin.Context, body []byte, model string) error {
	resp, responseStart, err := forwardRequest(c, body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to forward request"})
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "non_stream", "failed").Inc()
		return err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	responseDuration := float64(time.Since(responseStart)) / 1_000_000_000
	ollamaResponseSeconds.WithLabelValues(model, "ollama", "non_stream").Observe(responseDuration)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read response"})
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "non_stream", "failed").Inc()
		return fmt.Errorf("failed to read Ollama response: %v", err)
	}

	c.Writer.Write(responseBody)

	if resp.StatusCode != http.StatusOK {
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "non_stream", "failed").Inc()
		return fmt.Errorf("response was not success: status=%d", resp.StatusCode)
	}

	var responseData OllamaResponse
	if err := json.Unmarshal(responseBody, &responseData); err == nil {
		extractOllamaMetrics(&responseData, model)
		ollamaResponsesTotal.WithLabelValues(model, "ollama", "non_stream", "success").Inc()
	}
	return nil
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	proxyHandler, err := makeProxyHandler()
	if err != nil {
		log.Printf("Error: Can't create proxyHandler: %v", err)
	}

	if err := verifyOllamaConnection(); err != nil {
		log.Printf("Failed to connect to Ollama server: %v", err)
		log.Println("Please ensure Ollama is running and accessible at the configured host")
		os.Exit(1)
	}

	r := gin.Default()

	r.Use(gin.Recovery())

	r.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	r.POST("/api/chat", ollamaChatHandler)
	r.POST("/api/generate", ollamaChatHandler)
	r.POST("/v1/chat/completions", openAICompletionsHandler)

	r.NoRoute(proxyHandler)

	exporterAddr := envValue("EXPORTER_ADDR", "")
	port := envValue("EXPORTER_PORT", "8000")
	log.Printf("Ollama exporter listening at %s:%s", exporterAddr, port)
	if err := r.Run(exporterAddr + ":" + port); err != nil {
		log.Printf("Error starting server: %v", err)
		os.Exit(1)
	}
}

// envValue returns the value of an environment variable
func envValue(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// envDurationValue returns the value of an environment variable as a duration
func envDurationValue(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsedDuration, err := time.ParseDuration(value); err == nil {
			return parsedDuration
		}
		log.Printf("Warning: Invalid %s value '%s', using default '%s'.", key, value, defaultValue)
	}

	return defaultValue
}
