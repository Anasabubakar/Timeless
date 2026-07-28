package security

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// JWTKeyring supports rotating the JWT signing secret without
// invalidating every token issued under the previous one. New tokens are
// always signed with, and tagged (via the standard "kid" JWS header)
// with, the current key; verification looks the kid up among the current
// key plus any retired keys. Tokens issued before rotation support
// existed carry no kid at all, so lookups with an empty kid fall back to
// the current key — the same key those legacy tokens were actually
// signed with as long as JWTSecret itself hasn't changed.
type JWTKeyring struct {
	currentKeyID string
	keys         map[string][]byte
}

// NewJWTKeyring builds a keyring whose current key derives from secret.
// Pass previousSecrets (retired signing secrets, oldest first) to keep
// verifying tokens signed before a rotation.
func NewJWTKeyring(secret string, previousSecrets ...string) *JWTKeyring {
	kr := &JWTKeyring{keys: make(map[string][]byte, 1+len(previousSecrets))}

	currentKey := []byte(secret)
	currentID := jwtKeyID(currentKey)
	kr.keys[currentID] = currentKey
	kr.currentKeyID = currentID

	for _, prev := range previousSecrets {
		prev = strings.TrimSpace(prev)
		if prev == "" {
			continue
		}
		key := []byte(prev)
		kr.keys[jwtKeyID(key)] = key
	}

	return kr
}

func jwtKeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:4])
}

// CurrentKeyID is written into the "kid" header of every newly signed
// token.
func (kr *JWTKeyring) CurrentKeyID() string {
	return kr.currentKeyID
}

// CurrentKey is the secret newly issued tokens are signed with.
func (kr *JWTKeyring) CurrentKey() []byte {
	return kr.keys[kr.currentKeyID]
}

// Key resolves a kid (from a token's header) to a signing key. An empty
// kid — a token issued before rotation support existed — resolves to the
// current key, matching prior (pre-kid) behavior.
func (kr *JWTKeyring) Key(kid string) ([]byte, bool) {
	if kid == "" {
		return kr.CurrentKey(), true
	}
	key, ok := kr.keys[kid]
	return key, ok
}
