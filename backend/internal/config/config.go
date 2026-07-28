package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	Port        string `env:"PORT" envDefault:"8080"`
	FrontendURL string `env:"FRONTEND_URL" envDefault:"http://localhost:3000"`

	DatabaseURL string `env:"DATABASE_URL,required"`
	RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	JWTSecret     string        `env:"JWT_SECRET,required"`
	JWTExpiry     time.Duration `env:"JWT_EXPIRY" envDefault:"24h"`
	RefreshExpiry time.Duration `env:"REFRESH_EXPIRY" envDefault:"720h"`
	// JWTSecretPrevious holds retired JWT signing secrets (oldest first)
	// so rotating JWT_SECRET doesn't immediately invalidate every
	// access/refresh token already in a user's browser — see
	// security.JWTKeyring for how verification falls back through them.
	JWTSecretPrevious []string `env:"JWT_SECRET_PREVIOUS" envSeparator:","`
	// RememberMeExpiry is used instead of RefreshExpiry when the client
	// opts into "remember me" at login; RefreshExpiry alone doubles as the
	// non-remembered session length so existing behavior is unchanged.
	RememberMeExpiry time.Duration `env:"REMEMBER_ME_EXPIRY" envDefault:"2160h"` // 90 days

	EmailVerificationTTL time.Duration `env:"EMAIL_VERIFICATION_TTL" envDefault:"24h"`
	PasswordResetTTL     time.Duration `env:"PASSWORD_RESET_TTL" envDefault:"1h"`

	// Brute-force lockout: after MaxFailedLogins consecutive failures the
	// account is locked for LoginLockoutDuration. Counter resets on any
	// successful login.
	MaxFailedLogins      int           `env:"MAX_FAILED_LOGINS" envDefault:"5"`
	LoginLockoutDuration time.Duration `env:"LOGIN_LOCKOUT_DURATION" envDefault:"15m"`

	// MFAIssuer is the "issuer" label shown in authenticator apps
	// (Google Authenticator, 1Password, etc.) when a user enrolls TOTP.
	MFAIssuer string `env:"MFA_ISSUER" envDefault:"Timeless"`

	// CredentialsEncryptionKey is the secret integration credentials
	// (OAuth tokens, API keys) are encrypted with at rest. Defaults to
	// JWTSecret so no extra required env var exists out of the box, but a
	// dedicated key is recommended so rotating JWT signing doesn't force a
	// credential re-encryption pass too. CredentialsEncryptionKeyPrevious
	// holds retired keys (oldest first) so already-encrypted rows stay
	// readable through a rotation — see security.CredentialCipher.
	CredentialsEncryptionKey         string   `env:"CREDENTIALS_ENCRYPTION_KEY"`
	CredentialsEncryptionKeyPrevious []string `env:"CREDENTIALS_ENCRYPTION_KEY_PREVIOUS" envSeparator:","`

	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL"`

	// Public URL of this API, used to build OAuth redirect URIs.
	APIPublicURL string `env:"API_PUBLIC_URL" envDefault:"http://localhost:8080"`

	NotionClientID     string `env:"NOTION_CLIENT_ID"`
	NotionClientSecret string `env:"NOTION_CLIENT_SECRET"`
	ApolloClientID     string `env:"APOLLO_CLIENT_ID"`
	ApolloClientSecret string `env:"APOLLO_CLIENT_SECRET"`
	// Zapier has no OAuth app registration for third parties — it connects
	// via a personal MCP Server URL instead (see integration/zapier.go).

	OpenAIKey     string `env:"OPENAI_API_KEY"`
	AnthropicKey  string `env:"ANTHROPIC_API_KEY"`
	GeminiKey     string `env:"GEMINI_API_KEY"`
	GroqKey       string `env:"GROQ_API_KEY"`
	NvidiaKey     string `env:"NVIDIA_API_KEY"`
	NvidiaBaseURL string `env:"NVIDIA_BASE_URL" envDefault:"https://integrate.api.nvidia.com/v1"`
	OpenRouterKey string `env:"OPENROUTER_API_KEY"`

	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser     string `env:"SMTP_USER"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM"`
	SMTPFromName string `env:"SMTP_FROM_NAME" envDefault:"Timeless"`
	SMTPUseTLS   bool   `env:"SMTP_USE_TLS" envDefault:"false"`
	SendGridKey  string `env:"SENDGRID_API_KEY"`

	S3Endpoint  string `env:"S3_ENDPOINT"`
	S3Bucket    string `env:"S3_BUCKET" envDefault:"timeless"`
	S3AccessKey string `env:"S3_ACCESS_KEY"`
	S3SecretKey string `env:"S3_SECRET_KEY"`
	S3UseSSL    bool   `env:"S3_USE_SSL" envDefault:"false"`

	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`

	LogLevel  string `env:"LOG_LEVEL" envDefault:"debug"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"text"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CredentialKey is the secret to derive the credential encryption key
// from, falling back to JWTSecret when no dedicated key is configured.
func (c *Config) CredentialKey() string {
	if c.CredentialsEncryptionKey != "" {
		return c.CredentialsEncryptionKey
	}
	return c.JWTSecret
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
