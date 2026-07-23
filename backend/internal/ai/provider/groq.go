package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Groq struct {
	apiKey string
	client *http.Client
}

func NewGroq(apiKey string) *Groq {
	return &Groq{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (g *Groq) Name() string { return "groq" }

func (g *Groq) Models() []string {
	return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "gemma2-9b-it"}
}

func (g *Groq) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("groq marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("groq request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("groq do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("groq read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("groq error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
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

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("groq unmarshal: %w", err)
	}

	content := ""
	finishReason := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
	}

	return &CompletionResponse{
		Content:      content,
		Model:        result.Model,
		FinishReason: finishReason,
		TokensUsed: TokenUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

func (g *Groq) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		resp, err := g.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Content: fmt.Sprintf("error: %v", err), Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}
