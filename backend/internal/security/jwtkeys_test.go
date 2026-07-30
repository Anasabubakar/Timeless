package security

import "testing"

func TestJWTKeyringCurrentKeyResolvesByEmptyKid(t *testing.T) {
	kr := NewJWTKeyring("current-secret")
	key, ok := kr.Key("")
	if !ok {
		t.Fatal("expected an empty kid to resolve to the current key (pre-rotation tokens carry no kid at all)")
	}
	if string(key) != "current-secret" {
		t.Errorf("Key(\"\") = %q, want the current secret", key)
	}
}

func TestJWTKeyringCurrentKeyResolvesByItsOwnKid(t *testing.T) {
	kr := NewJWTKeyring("current-secret")
	key, ok := kr.Key(kr.CurrentKeyID())
	if !ok {
		t.Fatal("expected the current key's own kid to resolve")
	}
	if string(key) != "current-secret" {
		t.Errorf("Key(CurrentKeyID()) = %q, want the current secret", key)
	}
}

func TestJWTKeyringResolvesRetiredKeyByItsOwnKid(t *testing.T) {
	kr := NewJWTKeyring("current-secret", "retired-secret-one", "retired-secret-two")

	// A token signed under a retired key carries that key's own kid —
	// verification must still resolve it, or every outstanding token
	// signed before a rotation becomes invalid the instant JWT_SECRET
	// rotates, logging out every active session at once.
	retiredKr := NewJWTKeyring("retired-secret-one")
	retiredKid := retiredKr.CurrentKeyID()

	key, ok := kr.Key(retiredKid)
	if !ok {
		t.Fatal("expected a retired key's own kid to resolve")
	}
	if string(key) != "retired-secret-one" {
		t.Errorf("Key(retiredKid) = %q, want retired-secret-one", key)
	}
}

func TestJWTKeyringRejectsUnknownKid(t *testing.T) {
	kr := NewJWTKeyring("current-secret", "retired-secret")
	if _, ok := kr.Key("deadbeef"); ok {
		t.Error("expected an unrecognized kid to fail resolution — accepting it would mean any kid " +
			"value lets a forged token fall back to some key")
	}
}

func TestJWTKeyringCurrentKeyIDIsStableAndDeterministic(t *testing.T) {
	a := NewJWTKeyring("same-secret")
	b := NewJWTKeyring("same-secret")
	if a.CurrentKeyID() != b.CurrentKeyID() {
		t.Error("expected the same secret to always derive the same key id")
	}

	c := NewJWTKeyring("different-secret")
	if a.CurrentKeyID() == c.CurrentKeyID() {
		t.Error("expected different secrets to derive different key ids")
	}
}

func TestJWTKeyringEmptyKidAfterRotationStillMeansCurrent(t *testing.T) {
	// A pre-rotation-support token has no kid at all. After rotating,
	// Key("") must still resolve to the *current* key, matching prior
	// (pre-kid) behavior — not to whichever retired key happens to be
	// first in the list.
	kr := NewJWTKeyring("new-current-secret", "old-secret-1", "old-secret-2")
	key, ok := kr.Key("")
	if !ok || string(key) != "new-current-secret" {
		t.Errorf("Key(\"\") = %q, %v — want the current secret", key, ok)
	}
}

func TestJWTKeyringIgnoresBlankPreviousSecrets(t *testing.T) {
	// JWTSecretPrevious is comma-separated from an env var — a trailing
	// comma or accidental double-comma shouldn't register a blank
	// secret as a valid signing key.
	kr := NewJWTKeyring("current-secret", "", "  ", "retired-secret")
	blankKr := NewJWTKeyring("")
	if _, ok := kr.Key(blankKr.CurrentKeyID()); ok {
		t.Error("expected a blank previous secret to be skipped, not registered as a resolvable key")
	}
}
