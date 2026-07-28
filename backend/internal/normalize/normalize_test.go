package normalize

import "testing"

func TestDomain(t *testing.T) {
	cases := map[string]string{
		"Example.com":                "example.com",
		"https://www.Example.com/":   "example.com",
		"http://example.com/careers": "example.com",
		"example.com:8080":           "example.com",
		"  example.com  ":            "example.com",
		"www.example.com?utm=source": "example.com",
	}
	for in, want := range cases {
		if got := Domain(in); got != want {
			t.Errorf("Domain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmail(t *testing.T) {
	if got := Email("  Name@Example.COM  "); got != "name@example.com" {
		t.Errorf("Email() = %q, want name@example.com", got)
	}
}

func TestCompanyName(t *testing.T) {
	cases := map[string]string{
		"Acme Inc.":        "acme",
		"Acme, LLC":        "acme,",
		"Acme  Corp":       "acme",
		"  Acme  Widgets ": "acme widgets",
	}
	for in, want := range cases {
		if got := CompanyName(in); got != want {
			t.Errorf("CompanyName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompanyNameSameKeyForVariants(t *testing.T) {
	if CompanyName("Acme Inc.") != CompanyName("Acme") {
		t.Errorf("expected Acme Inc. and Acme to normalize to the same key")
	}
}
