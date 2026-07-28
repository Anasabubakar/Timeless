package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeZapierServer simulates a classic-mode MCP server: tools/list returns
// a fixed set of tools and no agentic meta-tools, so DiscoverApps must
// fall back to name-prefix grouping. It's TLS because zapierEndpoint only
// accepts an https:// mcp_server_url (the real mcp.zapier.com is https-only).
func fakeZapierServer(t *testing.T, onCall func(method string, params map[string]interface{})) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string                 `json:"method"`
			Params map[string]interface{} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if onCall != nil {
			onCall(req.Method, req.Params)
		}

		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "tools/list":
			w.Write([]byte(`{"result":{"tools":[
				{"name":"gmail_send_email","inputSchema":{}},
				{"name":"slack_find_message","inputSchema":{}}
			]}}`))
		case "tools/call":
			w.Write([]byte(`{"result":{"content":[{"text":"ok"}]}}`))
		default:
			w.Write([]byte(`{"result":{}}`))
		}
	}))
}

func TestDiscoverAppsClassicModeFallback(t *testing.T) {
	srv := fakeZapierServer(t, nil)
	defer srv.Close()

	c := &ZapierClient{httpClient: srv.Client()}
	apps, agentic, err := c.DiscoverApps(context.Background(), map[string]string{"mcp_server_url": srv.URL})
	if err != nil {
		t.Fatalf("DiscoverApps: %v", err)
	}
	if agentic {
		t.Errorf("expected classic mode (no meta-tools present), got agentic=true")
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 grouped apps (gmail_send, slack_find), got %v", apps)
	}
}

func TestExecuteActionSendsToolsCallWithArguments(t *testing.T) {
	var gotMethod string
	var gotParams map[string]interface{}
	srv := fakeZapierServer(t, func(method string, params map[string]interface{}) {
		if method == "tools/call" {
			gotMethod = method
			gotParams = params
		}
	})
	defer srv.Close()

	c := &ZapierClient{httpClient: srv.Client()}
	_, err := c.ExecuteAction(context.Background(), map[string]string{"mcp_server_url": srv.URL}, "gmail_send_email", map[string]interface{}{"to": "a@b.com"})
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if gotMethod != "tools/call" {
		t.Fatalf("expected a tools/call RPC, got %q", gotMethod)
	}
	if gotParams["name"] != "gmail_send_email" {
		t.Errorf("expected tool name gmail_send_email, got %v", gotParams["name"])
	}
	args, _ := gotParams["arguments"].(map[string]interface{})
	if args["to"] != "a@b.com" {
		t.Errorf("expected arguments to be passed through, got %v", args)
	}
}
