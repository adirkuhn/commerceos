package myparcel

import (
	"context"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// Adapter implements domain/integrations.MyParcelClient on top of Client,
// mapping domain-owned request/response shapes to this package's own HTTP
// types at the boundary -- same role as infrastructure/bol.Adapter.
type Adapter struct{ client *Client }

func NewAdapter(client *Client) *Adapter { return &Adapter{client: client} }

func (a *Adapter) CreateShipment(ctx context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
	r, err := a.client.CreateShipment(ctx, ShipmentInput{
		Carrier: in.Carrier,
		Recipient: Recipient{
			Person: in.Recipient.Person, Company: in.Recipient.Company,
			Street: in.Recipient.Street, Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix,
			PostalCode: in.Recipient.PostalCode, City: in.Recipient.City, CC: in.Recipient.CC,
			Email: in.Recipient.Email, Phone: in.Recipient.Phone,
		},
		Options: Options{
			PackageType: in.Options.PackageType, SignatureRequired: in.Options.SignatureRequired, OnlyRecipient: in.Options.OnlyRecipient,
			InsuranceAmountMinorUnits: insuranceAmountOrZero(in.Options.InsuranceAmountMinorEUR),
			InsuranceCurrency:         in.Options.Currency,
		},
		OrderReference: in.OrderReference,
		WeightGrams:    in.WeightGrams,
	})
	if err != nil {
		return integrationsDomain.ShipmentResult{}, err
	}
	return integrationsDomain.ShipmentResult{ShipmentID: r.ShipmentID, Barcode: r.Barcode}, nil
}

func (a *Adapter) DeleteShipment(ctx context.Context, myParcelShipmentID int64) error {
	return a.client.DeleteShipment(ctx, myParcelShipmentID)
}

func (a *Adapter) CreateOrder(ctx context.Context, in integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
	lines := make([]OrderLine, len(in.OrderLines))
	for i, l := range in.OrderLines {
		lines[i] = OrderLine{
			Quantity: l.Quantity, SKU: l.SKU, Name: l.Name,
			PriceMinorUnits: l.PriceMinorUnits, VATMinorUnits: l.VATMinorUnits, PriceAfterVATMinorUnits: l.PriceAfterVATMinorUnits,
		}
	}
	r, err := a.client.CreateOrder(ctx, OrderInput{
		ExternalIdentifier: in.ExternalIdentifier, OrderDate: in.OrderDate, AccountID: in.AccountID, ShopID: in.ShopID,
		Carrier: in.Carrier, PackageType: in.PackageType,
		Recipient: Recipient{
			Person: in.Recipient.Person, Company: in.Recipient.Company,
			Street: in.Recipient.Street, Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix,
			PostalCode: in.Recipient.PostalCode, City: in.Recipient.City, CC: in.Recipient.CC,
			Email: in.Recipient.Email, Phone: in.Recipient.Phone,
		},
		PriceMinorUnits: in.PriceMinorUnits, VATMinorUnits: in.VATMinorUnits, PriceAfterVATMinorUnits: in.PriceAfterVATMinorUnits,
		OrderLines: lines,
	})
	if err != nil {
		return integrationsDomain.OrderResult{}, err
	}
	return integrationsDomain.OrderResult{UUID: r.UUID, Status: r.Status}, nil
}

func insuranceAmountOrZero(amount *int) int {
	if amount == nil {
		return 0
	}
	return *amount
}

func (a *Adapter) GetLabel(ctx context.Context, myParcelShipmentID int64, format, size string) (integrationsDomain.LabelResult, error) {
	r, err := a.client.GetLabel(ctx, myParcelShipmentID, format, size)
	if err != nil {
		return integrationsDomain.LabelResult{}, err
	}
	return integrationsDomain.LabelResult{Data: r.Data, ContentType: r.ContentType}, nil
}

func (a *Adapter) GetTrackTrace(ctx context.Context, myParcelShipmentID int64) (integrationsDomain.TrackTraceResult, error) {
	r, err := a.client.GetTrackTrace(ctx, myParcelShipmentID)
	if err != nil {
		return integrationsDomain.TrackTraceResult{}, err
	}
	return integrationsDomain.TrackTraceResult{StatusCode: r.StatusCode, StatusMain: r.StatusMain, TrackingURL: r.TrackingURL}, nil
}

func (a *Adapter) GetShipmentBarcode(ctx context.Context, myParcelShipmentID int64) (string, error) {
	return a.client.GetShipmentBarcode(ctx, myParcelShipmentID)
}

func (a *Adapter) RegisterWebhook(ctx context.Context, callbackURL string) error {
	return a.client.RegisterWebhook(ctx, callbackURL)
}

func (a *Adapter) ListWebhookSubscriptions(ctx context.Context) ([]integrationsDomain.WebhookSubscription, error) {
	subs, err := a.client.ListWebhookSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WebhookSubscription, len(subs))
	for i, s := range subs {
		out[i] = integrationsDomain.WebhookSubscription{ID: s.ID, Hook: s.Hook, URL: s.URL}
	}
	return out, nil
}

func (a *Adapter) CreateWebhookSubscription(ctx context.Context, hook, deliveryURL string) (integrationsDomain.WebhookSubscription, error) {
	s, err := a.client.CreateWebhookSubscription(ctx, hook, deliveryURL)
	if err != nil {
		return integrationsDomain.WebhookSubscription{}, err
	}
	return integrationsDomain.WebhookSubscription{ID: s.ID, Hook: s.Hook, URL: s.URL}, nil
}

func (a *Adapter) DeleteWebhookSubscription(ctx context.Context, id string) error {
	return a.client.DeleteWebhookSubscription(ctx, id)
}

// ClientFactory implements domain/integrations.MyParcelClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(apiKey string) (integrationsDomain.MyParcelClient, error) {
	return NewAdapter(NewClient(Config{APIKey: apiKey})), nil
}
