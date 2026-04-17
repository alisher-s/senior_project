package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv string

	Server struct {
		Address string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	Postgres struct {
		Host     string
		Port     int
		User     string
		Password string
		Database string
		// SSLMode is passed to postgres DSN. In local dev use "disable".
		SSLMode string
	}

	Redis struct {
		Host     string
		Port     int
		Password string
		DB       int
	}

	JWT struct {
		AccessSecret  string
		RefreshSecret string
		AccessTTL     time.Duration
		RefreshTTL    time.Duration
		Issuer        string
		Audience      string
	}

	RateLimit struct {
		RequestsPerWindow int
		WindowSeconds     int
		// Max burst per second is derived from RequestsPerWindow/WindowSeconds (simple approach).
		// This is intentionally basic for the foundation; can be replaced with more advanced algorithms later.
	}

	Auth struct {
		NuEmailDomain string
		BcryptCost     int
	}

	Payments struct {
		// WebhookSecret is used to verify payment provider webhooks (HMAC-SHA256 over raw request body).
		WebhookSecret string
	}
}

// Load forwards to LoadFromEnv (legacy name used by some entrypoints).
func Load() (Config, error) {
	return LoadFromEnv()
}

func LoadFromEnv() (Config, error) {
	var cfg Config

	cfg.AppEnv = getenvDefault("APP_ENV", "development")

	// Server
	port := getenvDefault("PORT", "8080")
	cfg.Server.Address = ":" + port
	cfg.Server.ReadTimeout = getenvDurationDefault("SERVER_READ_TIMEOUT", 10*time.Second)
	cfg.Server.WriteTimeout = getenvDurationDefault("SERVER_WRITE_TIMEOUT", 10*time.Second)
	cfg.Server.IdleTimeout = getenvDurationDefault("SERVER_IDLE_TIMEOUT", 60*time.Second)

	// Postgres
	cfg.Postgres.Host = getenvDefault("POSTGRES_HOST", "postgres")
	cfg.Postgres.Port = getenvIntDefault("POSTGRES_PORT", 5432)
	cfg.Postgres.User = getenvDefault("POSTGRES_USER", "postgres")
	cfg.Postgres.Password = getenvDefault("POSTGRES_PASSWORD", "postgres")
	cfg.Postgres.Database = getenvDefault("POSTGRES_DB", "app")
	cfg.Postgres.SSLMode = getenvDefault("POSTGRES_SSLMODE", "disable")

	// Redis
	cfg.Redis.Host = getenvDefault("REDIS_HOST", "redis")
	cfg.Redis.Port = getenvIntDefault("REDIS_PORT", 6379)
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")
	cfg.Redis.DB = getenvIntDefault("REDIS_DB", 0)

	// JWT
	cfg.JWT.AccessSecret = getenvDefault("JWT_ACCESS_SECRET", "")
	cfg.JWT.RefreshSecret = getenvDefault("JWT_REFRESH_SECRET", "")
	cfg.JWT.AccessTTL = getenvDurationDefault("JWT_ACCESS_TTL", 15*time.Minute)
	cfg.JWT.RefreshTTL = getenvDurationDefault("JWT_REFRESH_TTL", 30*24*time.Hour)
	cfg.JWT.Issuer = getenvDefault("JWT_ISSUER", "nu-ticketing")
	cfg.JWT.Audience = getenvDefault("JWT_AUDIENCE", "nu-ticketing-client")

	// Rate limiting
	cfg.RateLimit.RequestsPerWindow = getenvIntDefault("RATE_LIMIT_REQUESTS", 120)
	cfg.RateLimit.WindowSeconds = getenvIntDefault("RATE_LIMIT_WINDOW_SECONDS", 60)

	// Auth
	cfg.Auth.NuEmailDomain = getenvDefault("AUTH_NU_EMAIL_DOMAIN", "nu.edu.kz")
	cfg.Auth.BcryptCost = getenvIntDefault("AUTH_BCRYPT_COST", 12)

	// Payments
	cfg.Payments.WebhookSecret = getenvDefault("PAYMENTS_WEBHOOK_SECRET", "")

	// Basic validation for required secrets
	if cfg.JWT.AccessSecret == "" && cfg.AppEnv == "development" {
		cfg.JWT.AccessSecret = "dev_access_secret_change_me"
	}
	if cfg.JWT.RefreshSecret == "" && cfg.AppEnv == "development" {
		cfg.JWT.RefreshSecret = "dev_refresh_secret_change_me"
	}
	if cfg.JWT.AccessSecret == "" {
		return cfg, fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if cfg.JWT.RefreshSecret == "" {
		return cfg, fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if cfg.Auth.BcryptCost < 4 {
		return cfg, fmt.Errorf("AUTH_BCRYPT_COST too low: %d", cfg.Auth.BcryptCost)
	}

	if cfg.Payments.WebhookSecret == "" && cfg.AppEnv == "development" {
		cfg.Payments.WebhookSecret = "dev_payments_webhook_secret_change_me"
	}
	if cfg.Payments.WebhookSecret == "" {
		return cfg, fmt.Errorf("PAYMENTS_WEBHOOK_SECRET is required")
	}

	return cfg, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

func getenvDurationDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}

