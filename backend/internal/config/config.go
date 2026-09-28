package config

import (
	"log"
	"net/url"
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
	// StorefrontPublicURL is where the storefront app is served (links in
	// shopper emails, e.g. newsletter confirmation). Defaults to
	// API_PUBLIC_BASE_URL's origin, where docker-compose mounts /store.
	StorefrontPublicURL string
	// MediaUploadDir is the local filesystem directory product images are
	// saved to (see infrastructure/storage) — served back out at
	// PublicAPIBaseURL + "/uploads/...". Only used when StorageDriver is
	// "local".
	MediaUploadDir string
	// StorageDriver selects the pms.FileStorage adapter: "local" (default)
	// or "s3" (MinIO / any S3-compatible store, see infrastructure/storage).
	StorageDriver string
	// S3* configure the S3-compatible client used when StorageDriver is
	// "s3". Only read/validated then (see cmd/api/main.go).
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3Bucket          string
	S3UseSSL          bool
	// S3PublicBaseURL is the publicly reachable URL prefix product images
	// are served from (bucket must allow public reads) — e.g. a domain
	// pointed at the MinIO/S3 endpoint or a CDN in front of it.
	S3PublicBaseURL string
	// LM2LicensePublicKeyURL is lm2's endpoint for fetching its current
	// RSA signing public key (see infrastructure/lm2license) -- used to
	// verify a POS desktop's license on every /pos/* request. Optional:
	// when unset, the /pos/* routes aren't mounted at all (see
	// cmd/api/main.go) rather than mounting a guaranteed-broken auth path,
	// same "optional integration, nil-safe skip" pattern as
	// AIDocumentParser being nil when no AI provider is configured.
	LM2LicensePublicKeyURL string
	// SMTP* configure the outbound-email client (infrastructure/messaging/smtp,
	// domain/messaging.Client) used to send storefront magic-link login
	// emails. Optional, same "unset means skip the whole feature rather
	// than mount a guaranteed-broken path" convention as
	// LM2LicensePublicKeyURL -- when SMTPHost is empty, cmd/api/main.go
	// doesn't mount the storefront auth routes at all.
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
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
		StorefrontPublicURL:          envOr("STOREFRONT_PUBLIC_URL", originOf(os.Getenv("API_PUBLIC_BASE_URL"))),
		MediaUploadDir:               envOr("MEDIA_UPLOAD_DIR", "./uploads"),
		StorageDriver:                envOr("STORAGE_DRIVER", "local"),
		S3Endpoint:                   os.Getenv("S3_ENDPOINT"),
		S3AccessKeyID:                os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:            os.Getenv("S3_SECRET_ACCESS_KEY"),
		S3Bucket:                     os.Getenv("S3_BUCKET"),
		S3UseSSL:                     envOr("S3_USE_SSL", "false") == "true",
		S3PublicBaseURL:              os.Getenv("S3_PUBLIC_BASE_URL"),
		LM2LicensePublicKeyURL:       os.Getenv("LM2_LICENSE_PUBLIC_KEY_URL"),
		SMTPHost:                     os.Getenv("SMTP_HOST"),
		SMTPPort:                     envOr("SMTP_PORT", "587"),
		SMTPUsername:                 os.Getenv("SMTP_USERNAME"),
		SMTPPassword:                 os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:                     os.Getenv("SMTP_FROM"),
	}
}

// originOf returns scheme://host of a URL, or "" when it can't be parsed.
func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
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
