package security

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPSecretIsUniqueAndValidBase32(t *testing.T) {
	a, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error: %v", err)
	}
	b, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error: %v", err)
	}
	if a == b {
		t.Error("expected two calls to produce different secrets")
	}
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(a); err != nil {
		t.Errorf("secret %q is not valid unpadded base32: %v", a, err)
	}
}

func TestTOTPProvisioningURI(t *testing.T) {
	uri := TOTPProvisioningURI("Timeless", "user@example.com", "JBSWY3DPEHPK3PXP")

	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Fatalf("expected an otpauth://totp/ URI, got %q", uri)
	}
	for _, want := range []string{
		"secret=JBSWY3DPEHPK3PXP",
		"issuer=Timeless",
		"algorithm=SHA1",
		"digits=6",
		"period=30",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("expected URI to contain %q, got %q", want, uri)
		}
	}
}

// TestValidateTOTPAcceptsTheCurrentCode computes the code the real
// function would compute for "right now" (via the unexported hotp,
// same math ValidateTOTP itself uses) and confirms ValidateTOTP accepts
// it — a live round-trip rather than a hardcoded vector that would
// eventually go stale relative to a real clock.
func TestValidateTOTPAcceptsTheCurrentCode(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error: %v", err)
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}

	counter := uint64(time.Now().Unix() / int64(totpPeriod.Seconds()))
	code := hotp(key, counter)

	if !ValidateTOTP(secret, code) {
		t.Errorf("expected the currently-valid code %q to be accepted for secret %q", code, secret)
	}
}

func TestValidateTOTPRejectsWrongCode(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error: %v", err)
	}
	if ValidateTOTP(secret, "000000") {
		t.Error("expected an almost-certainly-wrong fixed code to be rejected")
	}
}

func TestValidateTOTPRejectsWrongLength(t *testing.T) {
	secret, _ := GenerateTOTPSecret()
	for _, code := range []string{"", "12345", "1234567", "abcdef"} {
		if ValidateTOTP(secret, code) {
			t.Errorf("expected code %q (wrong length or non-numeric) to be rejected", code)
		}
	}
}

func TestValidateTOTPRejectsInvalidSecret(t *testing.T) {
	if ValidateTOTP("not-valid-base32!!!", "123456") {
		t.Error("expected an undecodable secret to reject any code rather than panicking or matching")
	}
}

func TestValidateTOTPToleratesClockSkew(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error: %v", err)
	}
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))

	// One period in the past — within the documented +/-1 skew window.
	pastCounter := uint64(time.Now().Add(-totpPeriod).Unix() / int64(totpPeriod.Seconds()))
	pastCode := hotp(key, pastCounter)
	if !ValidateTOTP(secret, pastCode) {
		t.Error("expected a code from one period ago to still be accepted within the skew window")
	}

	// Three periods in the past — outside the +/-1 skew window.
	farCounter := uint64(time.Now().Add(-3*totpPeriod).Unix() / int64(totpPeriod.Seconds()))
	farCode := hotp(key, farCounter)
	// Only assert rejection if the far-past counter actually differs
	// from every counter within the accepted window (guards against a
	// flaky assertion right at a period boundary).
	nowCounter := uint64(time.Now().Unix() / int64(totpPeriod.Seconds()))
	if farCounter < nowCounter-totpSkew || farCounter > nowCounter+totpSkew {
		if ValidateTOTP(secret, farCode) {
			t.Error("expected a code from three periods ago to be rejected — outside the skew window")
		}
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes(10)
	if err != nil {
		t.Fatalf("GenerateBackupCodes() error: %v", err)
	}
	if len(codes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(codes))
	}

	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate backup code generated: %q", code)
		}
		seen[code] = true

		if len(code) != 11 || code[5] != '-' { // 5 chars - 5 chars
			t.Errorf("backup code %q doesn't match the expected XXXXX-XXXXX shape", code)
		}
		for _, ambiguous := range []byte{'0', 'O', '1', 'I', 'L'} {
			if strings.IndexByte(code, ambiguous) >= 0 {
				t.Errorf("backup code %q contains an ambiguous character %q that should have been excluded", code, ambiguous)
			}
		}
	}
}
