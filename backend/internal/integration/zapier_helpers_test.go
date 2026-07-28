package integration

import "testing"

func TestToSortedApps(t *testing.T) {
	grouped := map[string]map[string]bool{
		"slack": {"slack_send_channel_message": true},
		"gmail": {"gmail_find_email": true, "gmail_send_email": true},
	}
	apps := toSortedApps(grouped)
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	if apps[0].Slug != "gmail" || apps[1].Slug != "slack" {
		t.Errorf("expected apps sorted alphabetically, got %v", apps)
	}
	if len(apps[0].Actions) != 2 {
		t.Errorf("expected gmail to have 2 actions, got %v", apps[0].Actions)
	}
}

func TestExtractActionListFromJSONArray(t *testing.T) {
	result := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"text": `["gmail_send_email","slack_send_channel_message"]`},
		},
	}
	actions := extractActionList(result)
	if len(actions) != 2 || actions[0] != "gmail_send_email" {
		t.Errorf("extractActionList() = %v, want [gmail_send_email slack_send_channel_message]", actions)
	}
}

func TestExtractActionListFromWrappedObject(t *testing.T) {
	result := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"text": `{"actions":["notion_create_page"]}`},
		},
	}
	actions := extractActionList(result)
	if len(actions) != 1 || actions[0] != "notion_create_page" {
		t.Errorf("extractActionList() = %v, want [notion_create_page]", actions)
	}
}

func TestExtractActionListEmpty(t *testing.T) {
	if actions := extractActionList(map[string]interface{}{}); len(actions) != 0 {
		t.Errorf("expected no actions from an empty result, got %v", actions)
	}
}

func TestModeLabel(t *testing.T) {
	if modeLabel(true) != "agentic" {
		t.Errorf("modeLabel(true) = %q, want agentic", modeLabel(true))
	}
	if modeLabel(false) != "classic" {
		t.Errorf("modeLabel(false) = %q, want classic", modeLabel(false))
	}
}

func TestSummarizeToolResultCapsAtThree(t *testing.T) {
	result := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"text": "one"},
			map[string]interface{}{"text": "two"},
			map[string]interface{}{"text": "three"},
			map[string]interface{}{"text": "four"},
		},
	}
	titles := summarizeToolResult("list_events", result)
	if len(titles) != 3 {
		t.Errorf("expected summarizeToolResult to cap at 3 titles, got %d: %v", len(titles), titles)
	}
}
