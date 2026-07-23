package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenAI struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{},
	}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Models() []string {
	return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}
}

func (o *OpenAI) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			}
		}
		body["tools"] = tools
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	choice := result.Choices[0]
	response := &CompletionResponse{
		Content:      choice.Message.Content,
		Model:        result.Model,
		FinishReason: choice.FinishReason,
		TokensUsed: TokenUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}

	for _, tc := range choice.Message.ToolCalls {
		response.ToolCalls = append(response.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return response, nil
}

func (o *OpenAI) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		req.Stream = true
		// Simplified: full SSE parsing would go here
		resp, err := o.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Content: fmt.Sprintf("error: %v", err), Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}

// Embed implements the Embedder interface for OpenAI
func (o *OpenAI) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	body := map[string]interface{}{
		"input": req.Input,
		"model": model,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embeddings error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Model string `json:"model"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      result.Model,
		TokensUsed: result.Usage.TotalTokens,
	}, nil
}
