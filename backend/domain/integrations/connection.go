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
	// ProviderStripe/ProviderRevolut are payment integrations -- StoreURL
	// unused (blank, same as Bol/MyParcel), ConsumerSecretEncrypted holds
	// the provider's secret/API key. See PaymentLinkClientFactory for the
	// payment integration contract both implement.
	ProviderStripe Provider = "stripe"
	// ProviderEmail is a per-tenant outbound SMTP relay -- reuses
	// domain/messaging.Client (infrastructure/messaging/smtp.Client) rather
	// than a dedicated factory port, since there's only one implementation
	// today (unlike Stripe/Revolut, competing implementations of one
	// PaymentLinkClient contract). Field mapping: StoreURL holds the SMTP
	// host (a bare hostname, not a URL -- StoreURL.Validate()'s http(s)
	// scheme check is NOT used for this provider), ConsumerKey the
	// username (plain, same "not the sensitive half" reasoning as Stripe's
	// publishable key), ConsumerSecretEncrypted the encrypted password,
	// SMTPPort/EmailFrom (below) the remaining two fields no existing
	// generic column covers. Before this provider existed, email was only
	// ever a single platform-wide SMTP relay configured via SMTP_HOST/etc
	// env vars (still the fallback cmd/api and cmd/scheduler use when no
	// tenant has connected one -- see those main.go files).
	ProviderEmail   Provider = "email"
	ProviderRevolut Provider = "revolut"
	// ProviderPostNL is a direct PostNL shipping-label integration (not
	// through MyParcel). ConsumerSecretEncrypted holds the PostNL API key;
	// PostNLCustomerCode/PostNLCustomerNumber/PostNLSandbox (below) the
	// account identifiers every PostNL call needs.
	ProviderPostNL Provider = "postnl"
	// ProviderMoneybird is the Moneybird bookkeeping integration.
	// ConsumerSecretEncrypted holds the encrypted personal API token; the
	// administration, mappings and auto-sync flag are its
	// domain/moneybird.Settings.
	ProviderMoneybird Provider = "moneybird"
)

// RequiresSenderAddress reports whether p's labels need the tenant's own
// sender address in every request (PostNL); MyParcel prints the sender
// from its own account settings.
func (p Provider) RequiresSenderAddress() bool {
	return p == ProviderPostNL
}

// IsPaymentProvider reports whether p is one of the payment integrations
// (Stripe/Revolut) -- used to filter which connections a SalesChannel's
// PaymentConnectionID may reference (see channels.SalesChannel's doc
// comment) and which are offered in the payment-method picker.
func (p Provider) IsPaymentProvider() bool {
	return p == ProviderStripe || p == ProviderRevolut
}

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
	ID       string
	TenantID string
	Provider Provider
	StoreURL StoreURL
	// ConsumerKey is provider-specific: WooCommerce's REST API key,
	// Bol's/WhatsApp's own instance identifier -- unused (blank) for
	// MyParcel (single credential). For a Stripe connection specifically,
	// it holds the publishable key ("pk_...") -- not sensitive, safe to
	// hand to the storefront client for Stripe Elements, so it's stored
	// plain here rather than encrypted like ConsumerSecretEncrypted.
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
	// BolShippingCostMinorUnits is what one parcel costs the merchant --
	// automatic Bol prices add it on top of the regular price (see
	// channels.ResolveBolPrices). 0 = none. Bol connections only.
	BolShippingCostMinorUnits int64
	// SMTPPort/EmailFrom are ProviderEmail-only (nil for every other
	// provider) -- see that constant's doc comment for the full field
	// mapping (StoreURL/ConsumerKey/ConsumerSecretEncrypted cover host/
	// username/password, these two cover the remaining pieces no generic
	// column already fit).
	SMTPPort  *string
	EmailFrom *string
	// WhatsAppTestModeEnabled/WhatsAppTestPhoneE164 are ProviderWhatsApp-only
	// (false/nil for every other provider). When enabled, every outbound
	// WhatsApp send through this connection is redirected to
	// WhatsAppTestPhoneE164 instead of the real recipient -- see
	// application/whatsapp's buildEvolutionClient, the single choke point
	// all 4 outbound-send call sites share. Lets a merchant test the
	// integration without risking a message reaching a real customer.
	WhatsAppTestModeEnabled bool
	WhatsAppTestPhoneE164   *string
	// WhatsAppAIParsingEnabled is ProviderWhatsApp-only (false for every
	// other provider). When true, a message that doesn't match the rigid
	// "COMMAND CODE" format is classified by the tenant's AI connection
	// (see whatsapp.AIIntentParser) instead of falling straight to help
	// text. SetWhatsAppAIParsingUseCase refuses to enable this unless the
	// tenant already has an AIConnection configured.
	WhatsAppAIParsingEnabled bool
	// PostNLCustomerCode (4 letters, e.g. "DEVC") and PostNLCustomerNumber
	// identify the PostNL business account labels are created for;
	// PostNLSandbox sends every call to PostNL's sandbox instead of
	// production. ProviderPostNL-only (empty/false for every other
	// provider).
	PostNLCustomerCode   string
	PostNLCustomerNumber string
	PostNLSandbox        bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
