package integrations

import (
	"context"
	"errors"

	"github.com/adirkuhn/commerceos/domain/errs"
)

// CarrierAddress is a shipment recipient's structured address, generic
// across shipping-label providers -- same shape MyParcelAddress has (every
// carrier that prints a label needs a street/number split), promoted out of
// being MyParcel-only so a second CarrierClient implementation doesn't need
// to reinvent it.
type CarrierAddress struct {
	Person       string
	Company      string
	Street       string
	Number       string
	NumberSuffix string
	PostalCode   string
	City         string
	CC           string
	Email        string
	Phone        string
}

// CarrierShipmentOptions mirrors MyParcelShipmentOptions' shape -- kept
// provider-agnostic (plain int/bool/pointer fields) so CreateShipmentUseCase
// never needs to know which concrete CarrierClient it's talking to.
type CarrierShipmentOptions struct {
	PackageType               int
	SignatureRequired         bool
	OnlyRecipient             bool
	InsuranceAmountMinorUnits *int
	Currency                  string
}

// CreateCarrierShipmentRequest is what CarrierClient.CreateShipment needs.
// Carrier is this codebase's own shippingDomain.Carrier value, passed as a
// plain int (domain packages don't import each other -- same reason
// CarrierAddress duplicates oms.Address's shape rather than importing it):
// the adapter, not the caller, is responsible for translating it into
// whatever its own provider's API expects.
type CreateCarrierShipmentRequest struct {
	Recipient CarrierAddress
	// Sender is the tenant's own company address (settings.CompanyProfile),
	// nil when none was saved. Carriers that print the sender from their
	// own account settings (MyParcel) ignore it; PostNL requires it.
	Sender *CarrierAddress
	// ReturnAddress is where the carrier sends a parcel it can't deliver;
	// nil means back to Sender.
	ReturnAddress  *CarrierAddress
	Carrier        int
	Options        CarrierShipmentOptions
	OrderReference string
	WeightGrams    *int
	// LabelFormat (pdf/zpl) is the label file format wanted -- only used by
	// carriers that return the label at creation (see
	// CarrierShipmentResult.Label).
	LabelFormat string
}

// CarrierShipmentResult is what CarrierClient.CreateShipment returns.
// ExternalShipmentID is a string, not MyParcelClient.ShipmentResult's int64
// ShipmentID -- a second carrier's own id format isn't guaranteed numeric.
// domain/shipping.Shipment.ExternalShipmentID (the persisted column) is
// this same generic string; see migration
// 160_shipments_external_shipment_id.sql.
type CarrierShipmentResult struct {
	ExternalShipmentID string
	Barcode            string
	// Label is set by carriers that hand the label over only once, in the
	// create response (PostNL), so CreateShipmentUseCase stores it;
	// nil for carriers whose label is fetched later through GetLabel.
	Label *LabelResult
}

// ErrCarrierOperationNotSupported: the connection's carrier has no API for
// this operation (e.g. PostNL can't cancel a manifested shipment or quote a
// delivery price).
var ErrCarrierOperationNotSupported = errs.New("CARRIER_OPERATION_NOT_SUPPORTED", "this carrier does not support this operation")

// ErrCarrierLabelNotDeletable: the carrier has no way to delete a
// pre-announced label (PostNL). It isn't billed until scanned, so
// cancelling just stops using it.
var ErrCarrierLabelNotDeletable = errors.New("carrier: pre-announced labels can't be deleted")

// ErrCarrierDestinationNotSupported: the carrier integration can't ship to
// the recipient's country (PostNL: outside NL, BE and its EU list -- those
// shipments need customs data).
var ErrCarrierDestinationNotSupported = errs.New("CARRIER_DESTINATION_NOT_SUPPORTED", "this carrier integration can't ship to this country yet")

// ErrCarrierSenderAddressMissing: the carrier needs the sender's address on
// every label and the company profile has none saved yet.
var ErrCarrierSenderAddressMissing = errs.New("CARRIER_SENDER_ADDRESS_MISSING", "fill in the company address in the company profile before creating labels with this carrier")

// CarrierClient is the shipping-label capability every carrier integration
// exposes: create/cancel a shipment, fetch its label/tracking/barcode, and
// price a delivery estimate. Deliberately excludes MyParcel's own
// Orders/Fulfilment export API (CreateOrder) and webhook-subscription
// management (RegisterWebhook/ListWebhookSubscriptions/
// CreateWebhookSubscription/DeleteWebhookSubscription) -- those stay on
// MyParcelClient directly (application/shipping.ExportMyParcelOrderUseCase,
// RegisterMyParcelWebhookUseCase, the provider-webhooks admin UI) since
// there is no second aggregator yet to validate a generic shape for them
// against; forcing one now would be guessing, not abstracting.
type CarrierClient interface {
	CreateShipment(ctx context.Context, in CreateCarrierShipmentRequest) (CarrierShipmentResult, error)
	// CancelShipment cancels a shipment by its ExternalShipmentID. Per
	// MyParcelClient.DeleteShipment's doc comment, a carrier may reject this
	// once a label already exists -- callers must treat that as an
	// informative outcome, not necessarily an application error.
	CancelShipment(ctx context.Context, externalShipmentID string) error
	GetLabel(ctx context.Context, externalShipmentID, format, size string) (LabelResult, error)
	GetTrackTrace(ctx context.Context, externalShipmentID string) (TrackTraceResult, error)
	// GetShipmentBarcode backfills a barcode outside of a status-change
	// webhook payload -- see MyParcelClient.GetShipmentBarcode's doc comment
	// for why this exists as its own call.
	GetShipmentBarcode(ctx context.Context, externalShipmentID string) (string, error)
	GetDeliveryPriceEstimate(ctx context.Context, in DeliveryPriceEstimateInput) ([]DeliveryPriceOption, error)
}

// CarrierCredentials is everything a CarrierClientFactory needs to build a
// client for one connection: which provider it is (so a multi-provider
// factory can dispatch to the right adapter) plus that provider's own
// credentials, already decrypted. Built by CarrierCredentialsFor.
type CarrierCredentials struct {
	Provider Provider
	APIKey   string
	// CustomerCode/CustomerNumber/Sandbox are PostNL's account identifiers
	// and environment switch (see Connection.PostNLCustomerCode); empty
	// for carriers that only need an API key.
	CustomerCode   string
	CustomerNumber string
	Sandbox        bool
}

// CarrierCredentialsFor builds conn's CarrierCredentials from its already-
// decrypted API key (ConsumerSecretEncrypted, decrypted by the caller --
// the domain never sees the encryption secret).
func CarrierCredentialsFor(conn Connection, apiKey string) CarrierCredentials {
	return CarrierCredentials{
		Provider: conn.Provider, APIKey: apiKey,
		CustomerCode: conn.PostNLCustomerCode, CustomerNumber: conn.PostNLCustomerNumber, Sandbox: conn.PostNLSandbox,
	}
}

// CarrierClientFactory builds a CarrierClient scoped to one connection's
// credentials -- the DI seam use cases depend on instead of a provider-
// specific factory (e.g. MyParcelClientFactory) once they only need this
// carrier-agnostic subset of capability. An implementation that can't build
// a client for creds.Provider returns ErrProviderNotSupported.
type CarrierClientFactory interface {
	NewClient(creds CarrierCredentials) (CarrierClient, error)
}
