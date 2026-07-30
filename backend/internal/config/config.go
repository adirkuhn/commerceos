package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDsn     string
	HTTPPort  string
	NatsURL   string
	JWTSecret string
	// IntegrationsEncryptionSecret encrypts stored third-party credentials
	// (e.g. WooCommerce consumer secrets) at rest — see internal/crypto.
	IntegrationsEncryptionSecret string
	// PublicAPIBaseURL is this API's own publicly reachable base URL, used to
	// build the callback_url WooCommerce's Application Authentication flow
	// POSTs back to (see application/integrations).
	PublicAPIBaseURL string
	// FrontendBaseURL is the CommerceOS frontend's base URL, used to build
	// the return_url the merchant's browser lands back on after approving.
	FrontendBaseURL string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		DBDsn:                        mustEnv("DB_DSN"),
		HTTPPort:                     envOr("HTTP_PORT", "8080"),
		NatsURL:                      mustEnv("NATS_URL"),
		JWTSecret:                    mustEnv("JWT_SECRET"),
		IntegrationsEncryptionSecret: mustEnv("INTEGRATIONS_ENCRYPTION_SECRET"),
		PublicAPIBaseURL:             mustEnv("API_PUBLIC_BASE_URL"),
		FrontendBaseURL:              mustEnv("FRONTEND_BASE_URL"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %q is not set", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
