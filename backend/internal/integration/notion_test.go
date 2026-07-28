package integration

import "testing"

func TestNotionTitle(t *testing.T) {
	properties := map[string]interface{}{
		"Tags": map[string]interface{}{"type": "multi_select"},
		"Name": map[string]interface{}{
			"type": "title",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Acme "},
				map[string]interface{}{"plain_text": "Corp"},
			},
		},
	}
	if got := notionTitle(properties); got != "Acme Corp" {
		t.Errorf("notionTitle() = %q, want %q", got, "Acme Corp")
	}
}

func TestNotionTitleNoTitleProperty(t *testing.T) {
	properties := map[string]interface{}{
		"Tags": map[string]interface{}{"type": "multi_select"},
	}
	if got := notionTitle(properties); got != "" {
		t.Errorf("notionTitle() with no title property = %q, want empty", got)
	}
}

func TestSplitName(t *testing.T) {
	cases := []struct {
		in          string
		first, last string
	}{
		{"", "", ""},
		{"Cher", "Cher", ""},
		{"Jane Doe", "Jane", "Doe"},
		{"Mary Jane Watson", "Mary", "Jane Watson"},
	}
	for _, c := range cases {
		first, last := splitName(c.in)
		if first != c.first || last != c.last {
			t.Errorf("splitName(%q) = (%q, %q), want (%q, %q)", c.in, first, last, c.first, c.last)
		}
	}
}

func TestFindPropertyByType(t *testing.T) {
	properties := map[string]interface{}{
		"Contact Email": map[string]interface{}{"type": "email"},
		"Name":          map[string]interface{}{"type": "title"},
	}
	if got := findPropertyByType(properties, "email"); got != "Contact Email" {
		t.Errorf("findPropertyByType(email) = %q, want %q", got, "Contact Email")
	}
	if got := findPropertyByType(properties, "phone_number"); got != "" {
		t.Errorf("findPropertyByType(phone_number) = %q, want empty", got)
	}
}

func TestFindPropertyLike(t *testing.T) {
	properties := map[string]interface{}{
		"Company Name": map[string]interface{}{},
		"Email":        map[string]interface{}{},
	}
	if got := findPropertyLike(properties, []string{"company", "organization"}); got != "Company Name" {
		t.Errorf("findPropertyLike(company) = %q, want %q", got, "Company Name")
	}
}
