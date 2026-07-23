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

	JWTSecret      string        `env:"JWT_SECRET,required"`
	JWTExpiry      time.Duration `env:"JWT_EXPIRY" envDefault:"24h"`
	RefreshExpiry  time.Duration `env:"REFRESH_EXPIRY" envDefault:"720h"`

	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL"`

	OpenAIKey     string `env:"OPENAI_API_KEY"`
	AnthropicKey  string `env:"ANTHROPIC_API_KEY"`
	GeminiKey     string `env:"GEMINI_API_KEY"`
	GroqKey       string `env:"GROQ_API_KEY"`
	OpenRouterKey string `env:"OPENROUTER_API_KEY"`

	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser     string `env:"SMTP_USER"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM"`
	SMTPFromName string `env:"SMTP_FROM_NAME" envDefault:"SponsorOS"`
	SMTPUseTLS   bool   `env:"SMTP_USE_TLS" envDefault:"false"`
	SendGridKey  string `env:"SENDGRID_API_KEY"`

	S3Endpoint  string `env:"S3_ENDPOINT"`
	S3Bucket    string `env:"S3_BUCKET" envDefault:"sponsoros"`
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

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
