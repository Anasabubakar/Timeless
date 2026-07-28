package integration

import "testing"

func TestAppSlugFromAction(t *testing.T) {
	cases := map[string]string{
		"google_calendar_create_event": "google_calendar",
		"gmail_send_email":             "gmail_send",
		"GoogleCalendar: Create Event": "googlecalendar",
		"slack":                        "slack",
	}
	for in, want := range cases {
		if got := appSlugFromAction(in); got != want {
			t.Errorf("appSlugFromAction(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGroupToolsByApp(t *testing.T) {
	tools := []mcpTool{
		{Name: "gmail_send_email"},
		{Name: "gmail_find_email"},
		{Name: "slack_send_channel_message"},
		{Name: toolListEnabledActions}, // meta-tool should never appear as its own "app"
	}

	apps := groupToolsByApp(tools)

	byName := make(map[string]ConnectedApp, len(apps))
	for _, a := range apps {
		byName[a.Slug] = a
	}

	gmail, ok := byName["gmail_send"]
	if !ok {
		t.Fatalf("expected a gmail_send app group, got %+v", apps)
	}
	if len(gmail.Actions) != 1 || gmail.Actions[0] != "gmail_send_email" {
		t.Errorf("gmail_send actions = %v, want [gmail_send_email]", gmail.Actions)
	}

	for _, a := range apps {
		for _, action := range a.Actions {
			if action == toolListEnabledActions {
				t.Errorf("meta-tool %q leaked into a grouped app", toolListEnabledActions)
			}
		}
	}
}

func TestIsSafeReadOnlyTool(t *testing.T) {
	cases := []struct {
		tool mcpTool
		want bool
	}{
		{mcpTool{Name: "gmail_find_email", InputSchema: map[string]interface{}{}}, true},
		{mcpTool{Name: "list_calendar_events", InputSchema: map[string]interface{}{}}, true},
		{mcpTool{Name: "gmail_send_email", InputSchema: map[string]interface{}{}}, false},
		{mcpTool{Name: "slack_delete_message", InputSchema: map[string]interface{}{}}, false},
		{mcpTool{Name: "gmail_find_email", InputSchema: map[string]interface{}{"required": []interface{}{"query"}}}, false},
		{mcpTool{Name: toolListEnabledActions, InputSchema: map[string]interface{}{}}, false},
	}
	for _, c := range cases {
		if got := isSafeReadOnlyTool(c.tool); got != c.want {
			t.Errorf("isSafeReadOnlyTool(%q) = %v, want %v", c.tool.Name, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short string should be unchanged, got %q", got)
	}
	if got := truncate("hello world", 5); got != "hello…" {
		t.Errorf("truncate(%q, 5) = %q, want %q", "hello world", got, "hello…")
	}
}
