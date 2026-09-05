package integrations

import (
	"net/url"
	"strings"
	"time"
)

type Provider string

const (
	ProviderWooCommerce Provider = "woocommerce"
	ProviderBol         Provider = "bol"
	ProviderMyParcel    Provider = "myparcel"
	// ProviderWhatsApp is a self-hosted Evolution API instance (unofficial,
	// Baileys-based WhatsApp Web gateway) -- StoreURL is the Evolution API
	// base URL, ConsumerKey its instance name, ConsumerSecretEncrypted its
	// instance API key. WebhookSecretEncrypted is the shared-secret path
	// token verified by infrastructure/webhookproviders' whatsapp verifier,
	// same convention as MyParcel (Evolution API signs nothing either).
	ProviderWhatsApp Provider = "whatsapp"
)

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
	// WebhookSecretEncrypted verifies INBOUND webhook signatures from this
	// provider -- distinct from ConsumerSecretEncrypted, which authenticates
	// our OUTBOUND REST calls to it. Empty until the tenant enables
	// webhooks for this connection (see application/webhooks package).
	WebhookSecretEncrypted string
	Status                 Status
	// DefaultCurrency is the store-level default used for product import
	// (WooCommerce's product endpoint doesn't return a currency — it's a
	// store-level setting the merchant configures here once, reused by
	// both the manual import trigger and the scheduled products/import
	// job, instead of being retyped on every run).
	DefaultCurrency string
	// DefaultWarehouseID is the connection's preferred CommerceOS warehouse,
	// nil when unset. It seeds a sync schedule row's own DefaultWarehouseID
	// (domain/channels.SyncSchedule) when that row hasn't picked one itself
	// -- same "set once here, reused everywhere" role as DefaultCurrency,
	// just for warehouse instead of currency.
	DefaultWarehouseID *string
	// BolEconomicOperatorID is Bol's required EU-operator identifier, sent
	// on every offer this connection pushes -- nil/empty for non-Bol
	// connections, which never read it.
	BolEconomicOperatorID *string
	// BolDeliveryCode is the delivery-promise enum Bol requires on every
	// FBR offer's fulfilment.deliveryCode -- nil means PushProductToBolUseCase
	// falls back to DefaultBolDeliveryCode ("3-5d"). Sitewide per connection,
	// not per-product -- unlike ConditionCategory/FulfilmentMethod there is
	// no BolProductOverride field for this by design (explicit user choice:
	// one delivery promise per connection is enough for now).
	BolDeliveryCode *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DefaultBolDeliveryCode is used whenever a Bol connection hasn't set
// BolDeliveryCode explicitly -- "3-5d" (ships in 3-5 days) is Bol's safe
// default for an FBR retailer with no same/next-day fulfilment capability.
const DefaultBolDeliveryCode = "3-5d"

// ValidBolDeliveryCodes is Bol's real Fulfilment.deliveryCode enum
// (Retailer API v10 spec) -- used to validate BolDeliveryCode at the
// settings-save boundary instead of only discovering a typo at push time.
var ValidBolDeliveryCodes = map[string]bool{
	"24uurs-23": true, "24uurs-22": true, "24uurs-21": true, "24uurs-20": true,
	"24uurs-19": true, "24uurs-18": true, "24uurs-17": true, "24uurs-16": true,
	"24uurs-15": true, "24uurs-14": true, "24uurs-13": true, "24uurs-12": true,
	"1-2d": true, "2-3d": true, "3-5d": true, "4-8d": true, "1-8d": true,
	"MijnLeverbelofte": true, "VVB": true,
}
