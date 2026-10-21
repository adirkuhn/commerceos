package myparcel

import (
	"context"
	"fmt"
	"strconv"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// CarrierAdapter adapts a MyParcelClient to the generic
// domain/integrations.CarrierClient contract -- the only place MyParcel's
// own wire shapes (int64 shipment ids, MyParcelAddress/
// MyParcelShipmentOptions) are translated to/from the provider-agnostic
// ones CreateShipmentUseCase and friends depend on.
type CarrierAdapter struct {
	inner integrationsDomain.MyParcelClient
}

func NewCarrierAdapter(inner integrationsDomain.MyParcelClient) *CarrierAdapter {
	return &CarrierAdapter{inner: inner}
}

var _ integrationsDomain.CarrierClient = (*CarrierAdapter)(nil)

func (a *CarrierAdapter) CreateShipment(ctx context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
	result, err := a.inner.CreateShipment(ctx, integrationsDomain.CreateShipmentInput{
		Recipient: integrationsDomain.MyParcelAddress{
			Person: in.Recipient.Person, Company: in.Recipient.Company, Street: in.Recipient.Street,
			Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix, PostalCode: in.Recipient.PostalCode,
			City: in.Recipient.City, CC: in.Recipient.CC, Email: in.Recipient.Email, Phone: in.Recipient.Phone,
		},
		Carrier: in.Carrier,
		Options: integrationsDomain.MyParcelShipmentOptions{
			PackageType: in.Options.PackageType, SignatureRequired: in.Options.SignatureRequired,
			OnlyRecipient: in.Options.OnlyRecipient, InsuranceAmountMinorEUR: in.Options.InsuranceAmountMinorUnits,
			Currency: in.Options.Currency,
		},
		OrderReference: in.OrderReference,
		WeightGrams:    in.WeightGrams,
	})
	if err != nil {
		return integrationsDomain.CarrierShipmentResult{}, err
	}
	return integrationsDomain.CarrierShipmentResult{
		ExternalShipmentID: strconv.FormatInt(result.ShipmentID, 10),
		Barcode:            result.Barcode,
	}, nil
}

func (a *CarrierAdapter) CancelShipment(ctx context.Context, externalShipmentID string) error {
	id, err := parseExternalShipmentID(externalShipmentID)
	if err != nil {
		return err
	}
	return a.inner.DeleteShipment(ctx, id)
}

func (a *CarrierAdapter) GetLabel(ctx context.Context, externalShipmentID, format, size string) (integrationsDomain.LabelResult, error) {
	id, err := parseExternalShipmentID(externalShipmentID)
	if err != nil {
		return integrationsDomain.LabelResult{}, err
	}
	return a.inner.GetLabel(ctx, id, format, size)
}

func (a *CarrierAdapter) GetTrackTrace(ctx context.Context, externalShipmentID string) (integrationsDomain.TrackTraceResult, error) {
	id, err := parseExternalShipmentID(externalShipmentID)
	if err != nil {
		return integrationsDomain.TrackTraceResult{}, err
	}
	return a.inner.GetTrackTrace(ctx, id)
}

func (a *CarrierAdapter) GetShipmentBarcode(ctx context.Context, externalShipmentID string) (string, error) {
	id, err := parseExternalShipmentID(externalShipmentID)
	if err != nil {
		return "", err
	}
	return a.inner.GetShipmentBarcode(ctx, id)
}

func (a *CarrierAdapter) GetDeliveryPriceEstimate(ctx context.Context, in integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
	return a.inner.GetDeliveryPriceEstimate(ctx, in)
}

func parseExternalShipmentID(externalShipmentID string) (int64, error) {
	id, err := strconv.ParseInt(externalShipmentID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("myparcel.CarrierAdapter: invalid external shipment id %q: %w", externalShipmentID, err)
	}
	return id, nil
}

// CarrierClientFactory builds a domain/integrations.CarrierClient backed by
// this package's own MyParcelClient -- the DI seam use cases depend on
// instead of ClientFactory directly, once they only need the
// carrier-agnostic subset of MyParcel's capability (see
// domain/integrations.CarrierClient's doc comment for what's excluded).
type CarrierClientFactory struct{ inner ClientFactory }

func NewCarrierClientFactory() CarrierClientFactory {
	return CarrierClientFactory{inner: NewClientFactory()}
}

var _ integrationsDomain.CarrierClientFactory = CarrierClientFactory{}

func (f CarrierClientFactory) NewClient(creds integrationsDomain.CarrierCredentials) (integrationsDomain.CarrierClient, error) {
	client, err := f.inner.NewClient(creds.APIKey)
	if err != nil {
		return nil, err
	}
	return NewCarrierAdapter(client), nil
}
