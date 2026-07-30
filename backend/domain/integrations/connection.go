package integrations

import (
	"net/url"
	"strings"
	"time"
)

type Provider string

const ProviderWooCommerce Provider = "woocommerce"

type Status string

const (
	StatusConnected Status = "connected"
	StatusError     Status = "error"
)

// StoreURL is the merchant's own WooCommerce store base URL, e.g. https://mystore.com.
type StoreURL string

func (s StoreURL) Validate() error {
	trimmed := strings.TrimSpace(string(s))
	if trimmed == "" {
		return ErrStoreURLInvalid
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrStoreURLInvalid
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrStoreURLInvalid
	}
	return nil
}

func (s StoreURL) String() string { return string(s) }

type Connection struct {
	ID                      string
	TenantID                string
	Provider                Provider
	StoreURL                StoreURL
	ConsumerKey             string
	ConsumerSecretEncrypted string
	Status                  Status
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
