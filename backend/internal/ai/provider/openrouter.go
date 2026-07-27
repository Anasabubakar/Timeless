package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenRouter struct {
	apiKey  string
	baseURL string
	client  *http.Client
	appName string
}

func NewOpenRouter(apiKey string) *OpenRouter {
	return &OpenRouter{
		apiKey:  apiKey,
		baseURL: "https://openrouter.ai/api/v1",
		client:  &http.Client{},
		appName: "Timeless",
	}
}

func (o *OpenRouter) Name() string { return "openrouter" }

func (o *OpenRouter) Models() []string {
	return []string{
		"anthropic/claude-sonnet-4-20250514",
		"openai/gpt-4o",
		"google/gemini-2.5-pro",
		"meta-llama/llama-3.3-70b-instruct",
		"mistralai/mistral-large-latest",
	}
}

func (o *OpenRouter) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "openai/gpt-4o-mini"
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
		return nil, fmt.Errorf("openrouter: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("openrouter: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://timeless.app")
	httpReq.Header.Set("X-Title", o.appName)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openrouter: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: status %d: %s", resp.StatusCode, string(respBody))
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

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("openrouter: unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openrouter: no choices in response")
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

	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			response.ToolCalls[i] = ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}

	return response, nil
}

func (o *OpenRouter) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 10)

	go func() {
		defer close(ch)
		resp, err := o.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		ch <- StreamChunk{
			Content: resp.Content,
			Done:    true,
		}
	}()

	return ch, nil
}
