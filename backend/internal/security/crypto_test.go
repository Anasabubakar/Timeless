package security

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := NewCredentialCipher("secret-v1")
	enc, err := c.Encrypt("hello world")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	plain, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if plain != "hello world" {
		t.Errorf("got %q, want %q", plain, "hello world")
	}
}

func TestLegacyUnprefixedCiphertextStillDecrypts(t *testing.T) {
	// Simulate data written before key-id tagging existed: raw base64,
	// no "keyid:" prefix.
	legacy := &CredentialCipher{keys: map[string][32]byte{legacyKeyID: deriveKey("secret-v1")}, currentKeyID: legacyKeyID}
	enc, err := seal(legacy.keys[legacyKeyID], "old secret")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	c := NewCredentialCipher("secret-v1")
	plain, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt legacy format: %v", err)
	}
	if plain != "old secret" {
		t.Errorf("got %q, want %q", plain, "old secret")
	}
}

func TestRotationKeepsOldDataReadable(t *testing.T) {
	oldCipher := NewCredentialCipher("secret-v1")
	enc, err := oldCipher.Encrypt("rotate me")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Rotate: new current key, old key kept as "previous".
	newCipher := NewCredentialCipher("secret-v2", "secret-v1")

	if !newCipher.NeedsRotation(enc) {
		t.Errorf("expected data encrypted under the retired key to need rotation")
	}

	plain, err := newCipher.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt data from retired key: %v", err)
	}
	if plain != "rotate me" {
		t.Errorf("got %q, want %q", plain, "rotate me")
	}

	// Re-encrypting under the new cipher should no longer need rotation.
	reEncrypted, err := newCipher.Encrypt(plain)
	if err != nil {
		t.Fatalf("re-Encrypt: %v", err)
	}
	if newCipher.NeedsRotation(reEncrypted) {
		t.Errorf("freshly re-encrypted data should not need rotation")
	}
}

func TestCurrentKeyIDMatchesEncryptTag(t *testing.T) {
	c := NewCredentialCipher("secret-v1")
	enc, err := c.Encrypt("value")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if got, want := enc[:len(c.CurrentKeyID())], c.CurrentKeyID(); got != want {
		t.Errorf("Encrypt() output isn't tagged with CurrentKeyID(): got prefix %q, want %q", got, want)
	}
}

func TestDecryptUnknownKeyIDFails(t *testing.T) {
	c := NewCredentialCipher("secret-v2", "secret-v1")
	otherCipher := NewCredentialCipher("secret-v3-never-registered")
	enc, err := otherCipher.Encrypt("orphaned")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := c.Decrypt(enc); err == nil {
		t.Errorf("expected decrypt to fail for a key id neither current nor previous")
	}
}

func TestStoredCredentialsRoundTrip(t *testing.T) {
	c := NewCredentialCipher("secret-v1")
	want := map[string]string{"token": "abc123", "refresh_token": "xyz789"}

	stored, err := c.EncryptStoredCredentials(want)
	if err != nil {
		t.Fatalf("EncryptStoredCredentials: %v", err)
	}
	got, err := c.DecryptStoredCredentials(stored)
	if err != nil {
		t.Fatalf("DecryptStoredCredentials: %v", err)
	}
	if got["token"] != want["token"] || got["refresh_token"] != want["refresh_token"] {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDecryptStoredCredentialsRejectsMalformedEnvelope(t *testing.T) {
	c := NewCredentialCipher("secret-v1")
	if _, err := c.DecryptStoredCredentials([]byte("not json")); err == nil {
		t.Error("expected an error for a malformed stored-credentials envelope")
	}
}

func TestStoredCredentialsSurvivesKeyRotation(t *testing.T) {
	old := NewCredentialCipher("secret-v1")
	stored, err := old.EncryptStoredCredentials(map[string]string{"token": "abc123"})
	if err != nil {
		t.Fatalf("EncryptStoredCredentials: %v", err)
	}

	rotated := NewCredentialCipher("secret-v2", "secret-v1")
	got, err := rotated.DecryptStoredCredentials(stored)
	if err != nil {
		t.Fatalf("DecryptStoredCredentials after rotation: %v", err)
	}
	if got["token"] != "abc123" {
		t.Errorf("got %v, want token=abc123", got)
	}
}
