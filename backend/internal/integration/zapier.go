package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ZapierClient connects to a user's personal Zapier MCP server
// (https://mcp.zapier.com), which speaks MCP over the Streamable HTTP
// transport (JSON-RPC 2.0). The "credential" is the full per-user MCP
// Server URL Zapier generates, which already embeds the auth secret —
// there is no separate API key to collect.
type ZapierClient struct {
	httpClient *http.Client
}

func NewZapierClient() *ZapierClient {
	return &ZapierClient{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *ZapierClient) Provider() string { return "zapier" }

const zapierDefaultMCPURL = "https://mcp.zapier.com/api/mcp/mcp"

// zapierEndpoint resolves how to reach the user's Zapier MCP server: an
// OAuth access token against the shared endpoint, or a personal MCP Server
// URL that embeds its own secret.
func zapierEndpoint(credentials map[string]string) (serverURL, bearer string, err error) {
	if token := strings.TrimSpace(credentials["access_token"]); token != "" {
		return zapierDefaultMCPURL, token, nil
	}
	serverURL = strings.TrimSpace(credentials["mcp_server_url"])
	if serverURL == "" || !strings.HasPrefix(serverURL, "https://") {
		return "", "", fmt.Errorf("mcp_server_url is required and must be the https MCP Server URL from mcp.zapier.com")
	}
	return serverURL, "", nil
}

func (c *ZapierClient) Validate(ctx context.Context, credentials map[string]string) error {
	serverURL, bearer, err := zapierEndpoint(credentials)
	if err != nil {
		return err
	}

	_, err = c.rpc(ctx, serverURL, bearer, "initialize", map[string]interface{}{
		"protocolVersion": "2026-06-18",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "timeless", "version": "1.0"},
	})
	if err != nil {
		return fmt.Errorf("could not reach Zapier MCP server: %w", err)
	}
	return nil
}

func (c *ZapierClient) Sync(ctx context.Context, credentials map[string]string) (*SyncResult, error) {
	serverURL, bearer, err := zapierEndpoint(credentials)
	if err != nil {
		return nil, err
	}
	result, err := c.rpc(ctx, serverURL, bearer, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	tools, _ := result["tools"].([]interface{})
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if tm, ok := t.(map[string]interface{}); ok {
			if name, ok := tm["name"].(string); ok {
				names = append(names, name)
			}
		}
	}

	return &SyncResult{
		Provider: c.Provider(),
		Details: map[string]interface{}{
			"available_tools": len(tools),
			"tool_names":      names,
		},
	}, nil
}

// rpc issues a single JSON-RPC 2.0 call over MCP's Streamable HTTP
// transport, which may respond with a plain JSON body or a single
// text/event-stream frame.
func (c *ZapierClient) rpc(ctx context.Context, serverURL, bearer, method string, params map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("zapier mcp server returned HTTP %d", resp.StatusCode)
	}

	payload, err := readMCPResponse(resp)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result map[string]interface{} `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode mcp response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcp error: %s", rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// readMCPResponse reads either a plain JSON body or a single "data: ..."
// frame from a text/event-stream response.
func readMCPResponse(resp *http.Response) ([]byte, error) {
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if data, ok := strings.CutPrefix(line, "data:"); ok {
			return []byte(strings.TrimSpace(data)), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("empty event-stream response")
}
