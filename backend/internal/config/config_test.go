package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_DSN", "dsn")
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("INTEGRATIONS_ENCRYPTION_SECRET", "enc-secret")
	t.Setenv("API_PUBLIC_BASE_URL", "https://api.example.com")
	t.Setenv("FRONTEND_BASE_URL", "https://app.example.com")
}

func TestLoad_defaultsStorageDriverToLocal(t *testing.T) {
	setRequiredEnv(t)

	cfg := Load()

	assert.Equal(t, "local", cfg.StorageDriver)
	assert.Equal(t, "./uploads", cfg.MediaUploadDir)
	assert.False(t, cfg.S3UseSSL)
	assert.Empty(t, cfg.S3Endpoint)
	assert.Empty(t, cfg.S3AccessKeyID)
	assert.Empty(t, cfg.S3SecretAccessKey)
	assert.Empty(t, cfg.S3Bucket)
	assert.Empty(t, cfg.S3PublicBaseURL)
}

func TestLoad_readsS3ConfigWhenStorageDriverIsS3(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("STORAGE_DRIVER", "s3")
	t.Setenv("S3_ENDPOINT", "minio:9000")
	t.Setenv("S3_ACCESS_KEY_ID", "minioadmin")
	t.Setenv("S3_SECRET_ACCESS_KEY", "minioadmin")
	t.Setenv("S3_BUCKET", "product-media")
	t.Setenv("S3_USE_SSL", "true")
	t.Setenv("S3_PUBLIC_BASE_URL", "https://media.example.com")

	cfg := Load()

	assert.Equal(t, "s3", cfg.StorageDriver)
	assert.Equal(t, "minio:9000", cfg.S3Endpoint)
	assert.Equal(t, "minioadmin", cfg.S3AccessKeyID)
	assert.Equal(t, "minioadmin", cfg.S3SecretAccessKey)
	assert.Equal(t, "product-media", cfg.S3Bucket)
	assert.True(t, cfg.S3UseSSL)
	assert.Equal(t, "https://media.example.com", cfg.S3PublicBaseURL)
}

func TestLoad_s3UseSSLDefaultsFalseForAnyNonTrueValue(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("S3_USE_SSL", "yes")

	cfg := Load()

	assert.False(t, cfg.S3UseSSL)
}
