// Package security provides at-rest encryption for small secrets (API keys,
// OAuth tokens) stored in JSONB columns such as Integration.Credentials.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// legacyKeyID tags ciphertext written before key-rotation support existed
// (no key-id prefix at all) and any pre-rotation data still encrypted under
// whatever secret was originally passed as the sole/current key.
const legacyKeyID = "legacy"

// CredentialCipher encrypts/decrypts secrets with AES-256-GCM. Encrypt
// always uses the current key and tags the ciphertext with that key's id;
// Decrypt looks the id up among the current key plus any retired
// (previous) keys, so rotating the active key doesn't break decryption of
// data written before the rotation. Unprefixed ciphertext (pre-dating
// rotation support entirely) falls back to the "legacy" key.
type CredentialCipher struct {
	currentKeyID string
	keys         map[string][32]byte
}

// NewCredentialCipher builds a cipher whose current key derives from
// secret. Pass previousSecrets (retired encryption secrets, oldest first)
// to keep decrypting data that was encrypted before a rotation — without
// them, rotating secret makes existing stored credentials unreadable.
func NewCredentialCipher(secret string, previousSecrets ...string) *CredentialCipher {
	c := &CredentialCipher{keys: make(map[string][32]byte, 1+len(previousSecrets))}

	currentKey := deriveKey(secret)
	currentID := deriveKeyID(currentKey)
	c.keys[currentID] = currentKey
	c.currentKeyID = currentID
	// The very first key this app ever used had no id prefix at all —
	// register it under "legacy" too so old rows still decrypt without
	// requiring an explicit CREDENTIALS_ENCRYPTION_KEY_PREVIOUS entry for
	// the common case for who's never rotated.
	c.keys[legacyKeyID] = currentKey

	for _, prev := range previousSecrets {
		prev = strings.TrimSpace(prev)
		if prev == "" {
			continue
		}
		k := deriveKey(prev)
		c.keys[deriveKeyID(k)] = k
	}

	return c
}

func deriveKey(secret string) [32]byte {
	return sha256.Sum256([]byte("credentials:" + secret))
}

func deriveKeyID(key [32]byte) string {
	sum := sha256.Sum256(key[:])
	return hex.EncodeToString(sum[:4])
}

// CurrentKeyID exposes which key new ciphertext is tagged with, so
// RotateAll-style maintenance code can tell whether a stored value is
// already on the current key without decrypting it.
func (c *CredentialCipher) CurrentKeyID() string {
	return c.currentKeyID
}

// NeedsRotation reports whether encoded was encrypted under something
// other than the current key (including the legacy unprefixed format).
func (c *CredentialCipher) NeedsRotation(encoded string) bool {
	keyID, _, ok := splitKeyTag(encoded)
	if !ok {
		return true // legacy format
	}
	return keyID != c.currentKeyID
}

func splitKeyTag(encoded string) (keyID, payload string, ok bool) {
	idx := strings.Index(encoded, ":")
	if idx <= 0 {
		return "", encoded, false
	}
	candidate := encoded[:idx]
	if len(candidate) != 8 { // hex.EncodeToString of 4 bytes
		return "", encoded, false
	}
	return candidate, encoded[idx+1:], true
}

func (c *CredentialCipher) Encrypt(plaintext string) (string, error) {
	key := c.keys[c.currentKeyID]
	ciphertext, err := seal(key, plaintext)
	if err != nil {
		return "", err
	}
	return c.currentKeyID + ":" + ciphertext, nil
}

func (c *CredentialCipher) Decrypt(encoded string) (string, error) {
	keyID, payload, tagged := splitKeyTag(encoded)
	if !tagged {
		keyID, payload = legacyKeyID, encoded
	}

	key, ok := c.keys[keyID]
	if !ok {
		return "", fmt.Errorf("unknown credential key id %q — was it rotated out without keeping the old secret in CREDENTIALS_ENCRYPTION_KEY_PREVIOUS?", keyID)
	}
	return open(key, payload)
}

// storedCredentials is the on-disk shape of an Integration.Credentials
// column: one AES-GCM-encrypted, key-tagged blob wrapping a JSON
// map[string]string of the provider's actual credential fields (token,
// refresh_token, ...).
type storedCredentials struct {
	Enc string `json:"enc"`
}

// DecryptStoredCredentials unwraps and decrypts an Integration.Credentials
// blob into the plain credential map — the same {"enc": "..."} envelope
// every credential-storing caller in this codebase writes via
// EncryptStoredCredentials.
func (c *CredentialCipher) DecryptStoredCredentials(raw []byte) (map[string]string, error) {
	var stored storedCredentials
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, fmt.Errorf("unmarshal stored credentials envelope: %w", err)
	}
	plain, err := c.Decrypt(stored.Enc)
	if err != nil {
		return nil, err
	}
	var credentials map[string]string
	if err := json.Unmarshal([]byte(plain), &credentials); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted credentials: %w", err)
	}
	return credentials, nil
}

// EncryptStoredCredentials is the inverse of DecryptStoredCredentials —
// encrypts a plain credential map and wraps it in the same envelope shape.
func (c *CredentialCipher) EncryptStoredCredentials(credentials map[string]string) ([]byte, error) {
	plain, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	enc, err := c.Encrypt(string(plain))
	if err != nil {
		return nil, err
	}
	return json.Marshal(storedCredentials{Enc: enc})
}

func seal(key [32]byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func open(key [32]byte, encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
