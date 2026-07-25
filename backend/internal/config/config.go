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
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		DBDsn:     mustEnv("DB_DSN"),
		HTTPPort:  envOr("HTTP_PORT", "8080"),
		NatsURL:   mustEnv("NATS_URL"),
		JWTSecret: mustEnv("JWT_SECRET"),
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
