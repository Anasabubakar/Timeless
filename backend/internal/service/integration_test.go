package service

import "testing"

func TestProviderType(t *testing.T) {
	if got := providerType("zapier"); got != "zapier" {
		t.Errorf("providerType(zapier) = %q, want %q", got, "zapier")
	}
	if got := providerType("notion"); got != "native" {
		t.Errorf("providerType(notion) = %q, want %q", got, "native")
	}
	if got := providerType("apollo"); got != "native" {
		t.Errorf("providerType(apollo) = %q, want %q", got, "native")
	}
}

func TestProviderName(t *testing.T) {
	cases := map[string]string{
		"zapier":  "Zapier",
		"notion":  "Notion",
		"apollo":  "Apollo",
		"unknown": "unknown",
	}
	for provider, want := range cases {
		if got := providerName(provider); got != want {
			t.Errorf("providerName(%q) = %q, want %q", provider, got, want)
		}
	}
}
