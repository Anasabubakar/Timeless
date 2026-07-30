package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

// TOTP implements RFC 6238 (TOTP) on top of RFC 4226 (HOTP) using only the
// standard library, so MFA doesn't pull in a third-party crypto dependency.
// Authenticator apps (Google Authenticator, Authy, 1Password, etc.) default
// to SHA1/30s/6-digit, so this package matches that for compatibility.
const (
	totpDigits = 6
	totpPeriod = 30 * time.Second
	// totpSkew allows the previous/next window to account for clock drift
	// between the server and the user's device.
	totpSkew = 1
)

// GenerateTOTPSecret returns a new random base32-encoded secret suitable
// for both HMAC-SHA1 TOTP generation and display/QR provisioning.
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20) // 160 bits, the RFC 4226 recommended HOTP secret length
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

// TOTPProvisioningURI builds an otpauth:// URI for QR-code enrollment.
func TOTPProvisioningURI(issuer, accountEmail, secret string) string {
	label := url.PathEscape(fmt.Sprintf("%s:%s", issuer, accountEmail))
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", totpDigits))
	q.Set("period", fmt.Sprintf("%d", int(totpPeriod.Seconds())))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode())
}

// ValidateTOTP checks a user-supplied code against the secret, allowing
// +/- totpSkew periods of clock drift. Uses constant-time comparison to
// avoid leaking timing information about which digits matched.
func ValidateTOTP(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	now := time.Now()
	for skew := -totpSkew; skew <= totpSkew; skew++ {
		counter := uint64(now.Add(time.Duration(skew)*totpPeriod).Unix() / int64(totpPeriod.Seconds()))
		expected := hotp(key, counter)
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func hotp(key []byte, counter uint64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)

	mod := uint32(1)
	for i := 0; i < totpDigits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", totpDigits, binCode%mod)
}

// GenerateBackupCodes returns n random alphanumeric recovery codes for use
// when the user's authenticator device is unavailable. Callers must hash
// them (HashBackupCode) before persisting.
func GenerateBackupCodes(n int) ([]string, error) {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789" // no ambiguous chars (0/O, 1/I/L)
	codes := make([]string, n)
	for i := range codes {
		var b strings.Builder
		for j := 0; j < 10; j++ {
			if j == 5 {
				b.WriteByte('-')
			}
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			if err != nil {
				return nil, err
			}
			b.WriteByte(alphabet[idx.Int64()])
		}
		codes[i] = b.String()
	}
	return codes, nil
}
