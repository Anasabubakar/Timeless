package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ZapierClient connects to Zapier MCP (https://mcp.zapier.com), which
// speaks MCP over the Streamable HTTP transport (JSON-RPC 2.0).
//
// Per Zapier's docs (docs.zapier.com/mcp/authenticate-with-zapier-mcp),
// a custom backend like ours authenticates with a "connection token" or
// "API key" generated from a server's Connect tab (choose "Other", or
// "TypeScript"/"Python" for the API key variant) and sent as a Bearer
// token against the fixed endpoint below — there is no per-user URL to
// parse. A legacy "mcp_server_url" credential is still honored for users
// who pasted a listed-client server URL instead.
//
// Zapier MCP servers run in one of two modes (docs.zapier.com/mcp, "Manage
// tools for your Zapier MCP server"):
//   - Classic mode: each enabled action is its own MCP tool, discoverable
//     only via tools/list.
//   - Agentic mode (beta): a small fixed set of meta-tools —
//     list_enabled_zapier_actions, discover_zapier_actions,
//     enable_zapier_action, execute_zapier_read_action,
//     execute_zapier_write_action — that let a caller enumerate and act on
//     the user's connected apps programmatically instead of parsing tool
//     name strings. Zapier is our PRIMARY integration gateway, so we prefer
//     agentic mode when it's available and fall back to classic tools/list
//     otherwise.
type ZapierClient struct {
	httpClient *http.Client
}

func NewZapierClient() *ZapierClient {
	return &ZapierClient{httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (c *ZapierClient) Provider() string { return "zapier" }

const zapierConnectURL = "https://mcp.zapier.com/api/v1/connect"

const (
	toolListEnabledActions = "list_enabled_zapier_actions"
	toolDiscoverActions    = "discover_zapier_actions"
)

// zapierEndpoint resolves how to reach the user's Zapier MCP server.
func zapierEndpoint(credentials map[string]string) (serverURL, bearer string, err error) {
	token := strings.TrimSpace(credentials["token"])
	if token == "" {
		token = strings.TrimSpace(credentials["access_token"])
	}
	if token != "" {
		return zapierConnectURL, token, nil
	}
	serverURL = strings.TrimSpace(credentials["mcp_server_url"])
	if serverURL == "" || !strings.HasPrefix(serverURL, "https://") {
		return "", "", fmt.Errorf("token is required — generate one from mcp.zapier.com's Connect tab")
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

// mcpTool mirrors the shape of one entry in a tools/list response.
type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ConnectedApp is one third-party application discovered as available
// through the user's Zapier connection, with the actions we found for it.
type ConnectedApp struct {
	Slug    string   `json:"slug"`
	Actions []string `json:"actions"`
}

// DiscoverApps enumerates the applications reachable through the user's
// Zapier MCP connection. It prefers agentic mode's list_enabled_zapier_actions
// meta-tool (a real, structured list of enabled actions per app) and falls
// back to grouping classic-mode tools/list results by inferred app prefix
// when agentic mode isn't enabled for this server.
func (c *ZapierClient) DiscoverApps(ctx context.Context, credentials map[string]string) ([]ConnectedApp, bool, error) {
	serverURL, bearer, err := zapierEndpoint(credentials)
	if err != nil {
		return nil, false, err
	}

	tools, err := c.listTools(ctx, serverURL, bearer)
	if err != nil {
		return nil, false, err
	}

	if agentic, apps, ok := c.discoverAgentic(ctx, serverURL, bearer, tools); ok {
		return apps, agentic, nil
	}

	return groupToolsByApp(tools), false, nil
}

// discoverAgentic calls the agentic-mode meta-tools when present. It
// returns ok=false (not an error) when the server is running classic mode,
// so the caller falls back to tools/list grouping.
func (c *ZapierClient) discoverAgentic(ctx context.Context, serverURL, bearer string, tools []mcpTool) (agentic bool, apps []ConnectedApp, ok bool) {
	hasMetaTool := false
	for _, t := range tools {
		if t.Name == toolListEnabledActions || t.Name == toolDiscoverActions {
			hasMetaTool = true
			break
		}
	}
	if !hasMetaTool {
		return false, nil, false
	}

	result, err := c.callTool(ctx, serverURL, bearer, toolListEnabledActions, map[string]interface{}{})
	if err != nil {
		return true, nil, false
	}

	actions := extractActionList(result)
	if len(actions) == 0 {
		return true, nil, false
	}

	grouped := make(map[string]map[string]bool)
	for _, a := range actions {
		app := appSlugFromAction(a)
		if grouped[app] == nil {
			grouped[app] = map[string]bool{}
		}
		grouped[app][a] = true
	}
	return true, toSortedApps(grouped), true
}

func (c *ZapierClient) listTools(ctx context.Context, serverURL, bearer string) ([]mcpTool, error) {
	result, err := c.rpc(ctx, serverURL, bearer, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	raw, _ := result["tools"].([]interface{})
	tools := make([]mcpTool, 0, len(raw))
	for _, t := range raw {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		var tool mcpTool
		b, _ := json.Marshal(tm)
		if err := json.Unmarshal(b, &tool); err == nil && tool.Name != "" {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

// groupToolsByApp is the classic-mode fallback: infer the connected app
// from each tool's name prefix. Zapier doesn't document an exact slug
// format, so this is best-effort grouping for display purposes only — it
// never invents an app that isn't backed by a real tool.
func groupToolsByApp(tools []mcpTool) []ConnectedApp {
	grouped := make(map[string]map[string]bool)
	for _, t := range tools {
		if t.Name == toolListEnabledActions || t.Name == toolDiscoverActions {
			continue
		}
		app := appSlugFromAction(t.Name)
		if grouped[app] == nil {
			grouped[app] = map[string]bool{}
		}
		grouped[app][t.Name] = true
	}
	return toSortedApps(grouped)
}

func toSortedApps(grouped map[string]map[string]bool) []ConnectedApp {
	apps := make([]ConnectedApp, 0, len(grouped))
	for app, actionSet := range grouped {
		actions := make([]string, 0, len(actionSet))
		for a := range actionSet {
			actions = append(actions, a)
		}
		sort.Strings(actions)
		apps = append(apps, ConnectedApp{Slug: app, Actions: actions})
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Slug < apps[j].Slug })
	return apps
}

// appSlugFromAction guesses the source app from a tool/action name like
// "google_calendar_create_event" or "GoogleCalendar: Create Event" by
// taking everything up to the first known separator, falling back to the
// first token. This is a display heuristic, not a contract Zapier documents.
func appSlugFromAction(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.IndexAny(name, ":|"); idx > 0 {
		return normalizeSlug(name[:idx])
	}
	parts := strings.Split(name, "_")
	if len(parts) >= 2 {
		return normalizeSlug(parts[0] + "_" + parts[1])
	}
	if len(parts) == 1 {
		return normalizeSlug(parts[0])
	}
	return normalizeSlug(name)
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func extractActionList(result map[string]interface{}) []string {
	content, _ := result["content"].([]interface{})
	for _, c := range content {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		text, _ := cm["text"].(string)
		if text == "" {
			continue
		}
		var actions []string
		if err := json.Unmarshal([]byte(text), &actions); err == nil && len(actions) > 0 {
			return actions
		}
		var wrapped struct {
			Actions []string `json:"actions"`
		}
		if err := json.Unmarshal([]byte(text), &wrapped); err == nil && len(wrapped.Actions) > 0 {
			return wrapped.Actions
		}
	}
	if actions, ok := result["actions"].([]interface{}); ok {
		out := make([]string, 0, len(actions))
		for _, a := range actions {
			if s, ok := a.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// ExecuteAction invokes one Zapier tool/action by name (tools/call under
// the hood). This is the surface other services use to prefer Zapier over
// a native provider client — e.g. "create a calendar event" goes through
// here first, and only falls back to a native client if Zapier has no
// matching connected action.
func (c *ZapierClient) ExecuteAction(ctx context.Context, credentials map[string]string, actionName string, args map[string]interface{}) (map[string]interface{}, error) {
	serverURL, bearer, err := zapierEndpoint(credentials)
	if err != nil {
		return nil, err
	}
	return c.callTool(ctx, serverURL, bearer, actionName, args)
}

func (c *ZapierClient) callTool(ctx context.Context, serverURL, bearer, name string, args map[string]interface{}) (map[string]interface{}, error) {
	return c.rpc(ctx, serverURL, bearer, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
}

func (c *ZapierClient) Sync(ctx context.Context, credentials map[string]string, _ map[string]interface{}) (*SyncResult, error) {
	serverURL, bearer, err := zapierEndpoint(credentials)
	if err != nil {
		return nil, err
	}

	tools, err := c.listTools(ctx, serverURL, bearer)
	if err != nil {
		return nil, err
	}

	agentic, apps, ok := c.discoverAgentic(ctx, serverURL, bearer, tools)
	if !ok {
		apps = groupToolsByApp(tools)
	}

	appSlugs := make([]string, 0, len(apps))
	totalActions := 0
	for _, a := range apps {
		appSlugs = append(appSlugs, a.Slug)
		totalActions += len(a.Actions)
	}

	notes := c.safeReadOnlySync(ctx, serverURL, bearer, tools)

	return &SyncResult{
		Provider: c.Provider(),
		Details: map[string]interface{}{
			"mode":            modeLabel(agentic),
			"connected_apps":  appSlugs,
			"app_count":       len(apps),
			"available_tools": len(tools),
			"total_actions":   totalActions,
			"apps":            apps,
		},
		Notes: notes,
	}, nil
}

func modeLabel(agentic bool) string {
	if agentic {
		return "agentic"
	}
	return "classic"
}

var readOnlyVerbs = []string{"list_", "search_", "get_", "find_", "fetch_"}
var writeVerbs = []string{"send", "create", "delete", "update", "post", "remove", "write", "edit", "archive", "invite"}

// safeReadOnlySync opportunistically surfaces a handful of read-only,
// zero-required-argument tools (e.g. "list_calendar_events") as Activity
// notes, without ever invoking a tool that looks like it could mutate the
// user's connected apps (send/create/delete/... verbs are always skipped).
func (c *ZapierClient) safeReadOnlySync(ctx context.Context, serverURL, bearer string, tools []mcpTool) []NoteRecord {
	notes := make([]NoteRecord, 0, 8)
	calls := 0
	for _, t := range tools {
		if calls >= 5 {
			break
		}
		if !isSafeReadOnlyTool(t) {
			continue
		}
		result, err := c.callTool(ctx, serverURL, bearer, t.Name, map[string]interface{}{})
		calls++
		if err != nil {
			continue
		}
		for _, title := range summarizeToolResult(t.Name, result) {
			notes = append(notes, NoteRecord{Title: title})
		}
	}
	return notes
}

func isSafeReadOnlyTool(t mcpTool) bool {
	lower := strings.ToLower(t.Name)
	if t.Name == toolListEnabledActions || t.Name == toolDiscoverActions {
		return false
	}
	for _, w := range writeVerbs {
		if strings.Contains(lower, w) {
			return false
		}
	}
	isRead := false
	for _, r := range readOnlyVerbs {
		if strings.HasPrefix(lower, r) || strings.Contains(lower, "_"+r) {
			isRead = true
			break
		}
	}
	if !isRead {
		return false
	}
	required, _ := t.InputSchema["required"].([]interface{})
	return len(required) == 0
}

func summarizeToolResult(toolName string, result map[string]interface{}) []string {
	content, _ := result["content"].([]interface{})
	titles := make([]string, 0, 3)
	for _, c := range content {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		text, _ := cm["text"].(string)
		if text == "" {
			continue
		}
		titles = append(titles, fmt.Sprintf("%s: %s", toolName, truncate(text, 140)))
		if len(titles) >= 3 {
			break
		}
	}
	return titles
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
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

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, &RateLimitError{Provider: "zapier", RetryAfter: retryAfter}
	}
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

// maxMCPLineSize overrides bufio.Scanner's 64KB default token limit. A real
// tools/list response listing every enabled Zapier action (each with a
// full JSON input schema) for a workspace with many connected apps easily
// exceeds that default and fails with "bufio.Scanner: token too long" —
// confirmed against a live Zapier MCP server, not a hypothetical.
const maxMCPLineSize = 10 << 20 // 10MB

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
	scanner.Buffer(make([]byte, 0, 64*1024), maxMCPLineSize)
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
