package handlers

// OllamaResponse mirrors the HTTP response body for Ollama completions.
type OllamaResponse struct {
	TotalDuration      int64 `json:"total_duration"`
	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	PromptEvalCount    int64 `json:"prompt_eval_count"`
	EvalDuration       int64 `json:"eval_duration"`
	EvalCount          int64 `json:"eval_count"`
	Done               bool  `json:"done"`
}

// OllamaChunk is the per-line chunk seen in streaming responses.
type OllamaChunk struct {
	TotalDuration      int64 `json:"total_duration"`
	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	PromptEvalCount    int64 `json:"prompt_eval_count"`
	EvalDuration       int64 `json:"eval_duration"`
	EvalCount          int64 `json:"eval_count"`
	Done               bool  `json:"done"`
}
