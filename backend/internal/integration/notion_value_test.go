package integration

import "testing"

func TestNotionPlainValueEmail(t *testing.T) {
	prop := map[string]interface{}{"type": "email", "email": "jane@example.com"}
	if got := notionPlainValue(prop); got != "jane@example.com" {
		t.Errorf("notionPlainValue(email) = %q, want %q", got, "jane@example.com")
	}
}

func TestNotionPlainValueSelect(t *testing.T) {
	prop := map[string]interface{}{
		"type":   "select",
		"select": map[string]interface{}{"name": "Enterprise"},
	}
	if got := notionPlainValue(prop); got != "Enterprise" {
		t.Errorf("notionPlainValue(select) = %q, want %q", got, "Enterprise")
	}
}

func TestNotionPlainValueUnknownType(t *testing.T) {
	prop := map[string]interface{}{"type": "checkbox", "checkbox": true}
	if got := notionPlainValue(prop); got != "" {
		t.Errorf("notionPlainValue(checkbox) = %q, want empty (unsupported type)", got)
	}
}

func TestRichTextPlainConcatenates(t *testing.T) {
	parts := []interface{}{
		map[string]interface{}{"plain_text": "Hello, "},
		map[string]interface{}{"plain_text": "world"},
	}
	if got := richTextPlain(parts); got != "Hello, world" {
		t.Errorf("richTextPlain() = %q, want %q", got, "Hello, world")
	}
}

func TestRichTextPlainNonArray(t *testing.T) {
	if got := richTextPlain("not an array"); got != "" {
		t.Errorf("richTextPlain() on a non-array = %q, want empty", got)
	}
}
