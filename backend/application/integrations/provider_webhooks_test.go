package integrations_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// fakeFullWooClient implements the entire integrations.WooClient interface
// (required for use as a WooClientFactory's return value) but only the
// webhook methods are exercised by these tests -- every other method
// panics if called, same convention as mocks.OutboxRepo's unused methods.
type fakeFullWooClient struct {
	listWebhooksFn  func(ctx context.Context) ([]integrationsDomain.WooWebhook, error)
	createWebhookFn func(ctx context.Context, topic, deliveryURL, secret string) (integrationsDomain.WooWebhook, error)
	deleteWebhookFn func(ctx context.Context, id string) error
}

func (f *fakeFullWooClient) TestConnection(ctx context.Context) error { panic("not used") }
func (f *fakeFullWooClient) ListProducts(ctx context.Context, page, perPage int) ([]integrationsDomain.WooProduct, error) {
	panic("not used")
}
func (f *fakeFullWooClient) ListProductStock(context.Context, int, int) ([]integrationsDomain.WooStockSnapshot, error) {
	panic("not used")
}
func (f *fakeFullWooClient) ListVariationStock(context.Context, string, int, int) ([]integrationsDomain.WooStockSnapshot, error) {
	panic("not used")
}
func (f *fakeFullWooClient) CreateProduct(ctx context.Context, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	panic("not used")
}
func (f *fakeFullWooClient) UpdateProductStock(ctx context.Context, parentExternalID, externalID string, quantity int, metadata []integrationsDomain.WooMetaData) error {
	panic("not used")
}
func (f *fakeFullWooClient) UpdateProduct(ctx context.Context, externalProductID string, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	panic("not used")
}
func (f *fakeFullWooClient) FindProductBySKU(ctx context.Context, sku string) (*integrationsDomain.WooProduct, error) {
	panic("not used")
}
func (f *fakeFullWooClient) FindTrashedProductBySKU(ctx context.Context, sku string) (*integrationsDomain.WooProduct, error) {
	panic("not used")
}
func (f *fakeFullWooClient) CreateVariableProduct(ctx context.Context, p integrationsDomain.WooProduct, attributeName string, attributeValues []string) (*integrationsDomain.WooProduct, error) {
	panic("not used")
}
func (f *fakeFullWooClient) CreateVariation(ctx context.Context, parentExternalID string, v integrationsDomain.WooVariation) (*integrationsDomain.WooVariation, error) {
	panic("not used")
}
func (f *fakeFullWooClient) UpdateVariation(ctx context.Context, parentExternalID, variationExternalID string, v integrationsDomain.WooVariation) (*integrationsDomain.WooVariation, error) {
	panic("not used")
}
func (f *fakeFullWooClient) DeleteVariation(ctx context.Context, parentExternalID, variationExternalID string) error {
	panic("not used")
}
func (f *fakeFullWooClient) DeleteProduct(ctx context.Context, externalProductID string) error {
	panic("not used")
}
func (f *fakeFullWooClient) ListCategories(ctx context.Context) ([]integrationsDomain.WooCategory, error) {
	panic("not used")
}
func (f *fakeFullWooClient) GetProductUnits(ctx context.Context) (string, string, error) {
	panic("not used")
}
func (f *fakeFullWooClient) DownloadImage(ctx context.Context, url string) (io.ReadCloser, string, error) {
	panic("not used")
}
func (f *fakeFullWooClient) ListOrderStatuses(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (f *fakeFullWooClient) ListOrders(ctx context.Context, page, perPage int, modifiedAfter *time.Time) ([]integrationsDomain.WooOrder, error) {
	panic("not used")
}
func (f *fakeFullWooClient) GetOrder(ctx context.Context, externalOrderID string) (*integrationsDomain.WooOrder, error) {
	panic("not used")
}
func (f *fakeFullWooClient) UpdateOrderStatus(ctx context.Context, externalOrderID, status string) error {
	panic("not used")
}
func (f *fakeFullWooClient) AddOrderNote(ctx context.Context, externalOrderID, note string) error {
	panic("not used")
}
func (f *fakeFullWooClient) ListCustomers(ctx context.Context, page, perPage int) ([]integrationsDomain.WooCustomer, error) {
	panic("not used")
}
func (f *fakeFullWooClient) ListTaxRates(ctx context.Context) ([]integrationsDomain.WooTaxRate, error) {
	panic("not used")
}
func (f *fakeFullWooClient) ListWebhooks(ctx context.Context) ([]integrationsDomain.WooWebhook, error) {
	return f.listWebhooksFn(ctx)
}
func (f *fakeFullWooClient) CreateWebhook(ctx context.Context, topic, deliveryURL, secret string) (integrationsDomain.WooWebhook, error) {
	return f.createWebhookFn(ctx, topic, deliveryURL, secret)
}
func (f *fakeFullWooClient) DeleteWebhook(ctx context.Context, id string) error {
	return f.deleteWebhookFn(ctx, id)
}

// fakeFullMyParcelClient mirrors fakeFullWooClient's role for
// integrations.MyParcelClient.
type fakeFullMyParcelClient struct {
	listWebhookSubscriptionsFn  func(ctx context.Context) ([]integrationsDomain.WebhookSubscription, error)
	createWebhookSubscriptionFn func(ctx context.Context, hook, deliveryURL string) (integrationsDomain.WebhookSubscription, error)
	deleteWebhookSubscriptionFn func(ctx context.Context, id string) error
}

func (f *fakeFullMyParcelClient) CreateShipment(ctx context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
	panic("not used")
}
func (f *fakeFullMyParcelClient) DeleteShipment(ctx context.Context, myParcelShipmentID int64) error {
	panic("not used")
}
func (f *fakeFullMyParcelClient) GetDeliveryPriceEstimate(ctx context.Context, in integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
	panic("not used")
}
func (f *fakeFullMyParcelClient) CreateOrder(ctx context.Context, in integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
	panic("not used")
}
func (f *fakeFullMyParcelClient) GetLabel(ctx context.Context, myParcelShipmentID int64, format, size string) (integrationsDomain.LabelResult, error) {
	panic("not used")
}
func (f *fakeFullMyParcelClient) GetTrackTrace(ctx context.Context, myParcelShipmentID int64) (integrationsDomain.TrackTraceResult, error) {
	panic("not used")
}
func (f *fakeFullMyParcelClient) GetShipmentBarcode(ctx context.Context, myParcelShipmentID int64) (string, error) {
	panic("not used")
}
func (f *fakeFullMyParcelClient) RegisterWebhook(ctx context.Context, callbackURL string) error {
	panic("not used")
}
func (f *fakeFullMyParcelClient) ListWebhookSubscriptions(ctx context.Context) ([]integrationsDomain.WebhookSubscription, error) {
	return f.listWebhookSubscriptionsFn(ctx)
}
func (f *fakeFullMyParcelClient) CreateWebhookSubscription(ctx context.Context, hook, deliveryURL string) (integrationsDomain.WebhookSubscription, error) {
	return f.createWebhookSubscriptionFn(ctx, hook, deliveryURL)
}
func (f *fakeFullMyParcelClient) DeleteWebhookSubscription(ctx context.Context, id string) error {
	return f.deleteWebhookSubscriptionFn(ctx, id)
}

type fakeMyParcelClientFactory struct {
	client integrationsDomain.MyParcelClient
	err    error
}

func (f *fakeMyParcelClientFactory) NewClient(apiKey string) (integrationsDomain.MyParcelClient, error) {
	return f.client, f.err
}

func wooConn(overrides func(*integrationsDomain.Connection)) *integrationsDomain.Connection {
	secret, _ := crypto.Encrypt(testEncSecret, "consumer-secret")
	c := &integrationsDomain.Connection{
		ID: "conn-1", TenantID: "tenant-1", Provider: integrationsDomain.ProviderWooCommerce,
		StoreURL: "https://mystore.com", ConsumerKey: "ck", ConsumerSecretEncrypted: secret,
	}
	if overrides != nil {
		overrides(c)
	}
	return c
}

func myParcelConn(overrides func(*integrationsDomain.Connection)) *integrationsDomain.Connection {
	secret, _ := crypto.Encrypt(testEncSecret, "api-key")
	c := &integrationsDomain.Connection{
		ID: "conn-1", TenantID: "tenant-1", Provider: integrationsDomain.ProviderMyParcel,
		ConsumerSecretEncrypted: secret,
	}
	if overrides != nil {
		overrides(c)
	}
	return c
}

const testEncSecret = "test-encryption-secret-32-bytes"

func TestListProviderWebhooksUseCase_woo_listsFromLiveStore(t *testing.T) {
	conn := wooConn(nil)
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	wooClient := &fakeFullWooClient{listWebhooksFn: func(_ context.Context) ([]integrationsDomain.WooWebhook, error) {
		return []integrationsDomain.WooWebhook{{ID: "1", Topic: "order.updated", DeliveryURL: "https://a.example.com", Status: "active"}}, nil
	}}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return wooClient, nil }}
	uc := appintegrations.NewListProviderWebhooksUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{}, testEncSecret)

	out, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "order.updated", out[0].Topic)
	assert.Equal(t, integrationsDomain.ProviderWooCommerce, out[0].Provider)
}

func TestListProviderWebhooksUseCase_myParcel_listsFromLiveAccount(t *testing.T) {
	conn := myParcelConn(nil)
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeFullMyParcelClient{listWebhookSubscriptionsFn: func(_ context.Context) ([]integrationsDomain.WebhookSubscription, error) {
		return []integrationsDomain.WebhookSubscription{{ID: "9", Hook: "shipment_status_change", URL: "https://a.example.com"}}, nil
	}}
	uc := appintegrations.NewListProviderWebhooksUseCase(newMockGormDB(t), connRepo, &mocks.WooClientFactory{}, &fakeMyParcelClientFactory{client: client}, testEncSecret)

	out, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "shipment_status_change", out[0].Topic)
	assert.Equal(t, integrationsDomain.ProviderMyParcel, out[0].Provider)
}

func TestListProviderWebhooksUseCase_unsupportedProvider_returnsErrProviderNotSupported(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderBol}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	uc := appintegrations.NewListProviderWebhooksUseCase(newMockGormDB(t), connRepo, &mocks.WooClientFactory{}, &fakeMyParcelClientFactory{}, testEncSecret)

	_, err := uc.Execute(context.Background(), "conn-1")

	require.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}

func TestListProviderWebhooksUseCase_liveCallFailure_wrapsErrProviderWebhookRequestFailed(t *testing.T) {
	conn := wooConn(nil)
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	wooClient := &fakeFullWooClient{listWebhooksFn: func(_ context.Context) ([]integrationsDomain.WooWebhook, error) {
		return nil, errors.New("connection refused")
	}}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return wooClient, nil }}
	uc := appintegrations.NewListProviderWebhooksUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{}, testEncSecret)

	_, err := uc.Execute(context.Background(), "conn-1")

	require.ErrorIs(t, err, integrationsDomain.ErrProviderWebhookRequestFailed)
}

func TestListAllProviderWebhooksUseCase_aggregatesAcrossConnections(t *testing.T) {
	wc := wooConn(func(c *integrationsDomain.Connection) { c.ID = "conn-woo"; c.StoreURL = "https://mystore.com" })
	mp := myParcelConn(func(c *integrationsDomain.Connection) { c.ID = "conn-mp" })
	connRepo := &mocks.ConnectionRepo{ListByTenantFn: func(_ context.Context) ([]integrationsDomain.Connection, error) {
		return []integrationsDomain.Connection{*wc, *mp}, nil
	}}
	wooClient := &fakeFullWooClient{listWebhooksFn: func(_ context.Context) ([]integrationsDomain.WooWebhook, error) {
		return []integrationsDomain.WooWebhook{{ID: "1", Topic: "order.updated", DeliveryURL: "https://a.example.com", Status: "active"}}, nil
	}}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return wooClient, nil }}
	mpClient := &fakeFullMyParcelClient{listWebhookSubscriptionsFn: func(_ context.Context) ([]integrationsDomain.WebhookSubscription, error) {
		return []integrationsDomain.WebhookSubscription{{ID: "9", Hook: "shipment_status_change", URL: "https://b.example.com"}}, nil
	}}
	uc := appintegrations.NewListAllProviderWebhooksUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{client: mpClient}, testEncSecret)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Len(t, result.Webhooks, 2)
	assert.Equal(t, "conn-woo", result.Webhooks[0].ConnectionID)
	assert.Equal(t, "https://mystore.com", result.Webhooks[0].StoreLabel)
	assert.Equal(t, "conn-mp", result.Webhooks[1].ConnectionID)
	assert.Equal(t, "MyParcel", result.Webhooks[1].StoreLabel)
}

func TestListAllProviderWebhooksUseCase_skipsUnsupportedProviders(t *testing.T) {
	bolConn := integrationsDomain.Connection{ID: "conn-bol", Provider: integrationsDomain.ProviderBol}
	connRepo := &mocks.ConnectionRepo{ListByTenantFn: func(_ context.Context) ([]integrationsDomain.Connection, error) {
		return []integrationsDomain.Connection{bolConn}, nil
	}}
	uc := appintegrations.NewListAllProviderWebhooksUseCase(newMockGormDB(t), connRepo, &mocks.WooClientFactory{}, &fakeMyParcelClientFactory{}, testEncSecret)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result.Webhooks)
	assert.Empty(t, result.Errors)
}

func TestListAllProviderWebhooksUseCase_oneConnectionFailing_stillReturnsTheRest(t *testing.T) {
	badWoo := wooConn(func(c *integrationsDomain.Connection) { c.ID = "conn-bad" })
	goodMP := myParcelConn(func(c *integrationsDomain.Connection) { c.ID = "conn-good" })
	connRepo := &mocks.ConnectionRepo{ListByTenantFn: func(_ context.Context) ([]integrationsDomain.Connection, error) {
		return []integrationsDomain.Connection{*badWoo, *goodMP}, nil
	}}
	badWooClient := &fakeFullWooClient{listWebhooksFn: func(_ context.Context) ([]integrationsDomain.WooWebhook, error) {
		return nil, errors.New("connection refused")
	}}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return badWooClient, nil }}
	goodMPClient := &fakeFullMyParcelClient{listWebhookSubscriptionsFn: func(_ context.Context) ([]integrationsDomain.WebhookSubscription, error) {
		return []integrationsDomain.WebhookSubscription{{ID: "1", Hook: "shipment_status_change", URL: "https://b.example.com"}}, nil
	}}
	uc := appintegrations.NewListAllProviderWebhooksUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{client: goodMPClient}, testEncSecret)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, result.Webhooks, 1)
	assert.Equal(t, "conn-good", result.Webhooks[0].ConnectionID)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "conn-bad")
}

func TestCreateProviderWebhookUseCase_woo_sendsIngestURLAndPlaintextSecret(t *testing.T) {
	plaintextSecret := "webhook-secret-plaintext"
	encryptedSecret, _ := crypto.Encrypt(testEncSecret, plaintextSecret)
	conn := wooConn(func(c *integrationsDomain.Connection) { c.WebhookSecretEncrypted = encryptedSecret })
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	var gotTopic, gotURL, gotSecret string
	wooClient := &fakeFullWooClient{createWebhookFn: func(_ context.Context, topic, deliveryURL, secret string) (integrationsDomain.WooWebhook, error) {
		gotTopic, gotURL, gotSecret = topic, deliveryURL, secret
		return integrationsDomain.WooWebhook{ID: "1", Topic: topic, DeliveryURL: deliveryURL, Status: "active"}, nil
	}}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return wooClient, nil }}
	uc := appintegrations.NewCreateProviderWebhookUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{}, testEncSecret, "https://api.example.com", &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "conn-1", "order.updated")

	require.NoError(t, err)
	assert.Equal(t, "order.updated", gotTopic)
	assert.Equal(t, "https://api.example.com/api/v1/webhooks/v1/tenant-1/woocommerce", gotURL)
	assert.Equal(t, plaintextSecret, gotSecret)
	assert.Equal(t, "1", result.ID)
}

func TestCreateProviderWebhookUseCase_myParcel_appendsTokenPathSegment(t *testing.T) {
	plaintextSecret := "webhook-token-plaintext"
	encryptedSecret, _ := crypto.Encrypt(testEncSecret, plaintextSecret)
	conn := myParcelConn(func(c *integrationsDomain.Connection) { c.WebhookSecretEncrypted = encryptedSecret })
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	var gotHook, gotURL string
	client := &fakeFullMyParcelClient{createWebhookSubscriptionFn: func(_ context.Context, hook, deliveryURL string) (integrationsDomain.WebhookSubscription, error) {
		gotHook, gotURL = hook, deliveryURL
		return integrationsDomain.WebhookSubscription{ID: "9", Hook: hook, URL: deliveryURL}, nil
	}}
	uc := appintegrations.NewCreateProviderWebhookUseCase(newMockGormDB(t), connRepo, &mocks.WooClientFactory{}, &fakeMyParcelClientFactory{client: client}, testEncSecret, "https://api.example.com", &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "conn-1", "shipment_status_change")

	require.NoError(t, err)
	assert.Equal(t, "shipment_status_change", gotHook)
	assert.Equal(t, "https://api.example.com/api/v1/webhooks/v1/tenant-1/myparcel/webhook-token-plaintext", gotURL)
	assert.Equal(t, "9", result.ID)
}

func TestCreateProviderWebhookUseCase_noWebhookSecretSet_returnsErrWebhookSecretNotSet(t *testing.T) {
	conn := wooConn(nil) // WebhookSecretEncrypted left empty
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	uc := appintegrations.NewCreateProviderWebhookUseCase(newMockGormDB(t), connRepo, &mocks.WooClientFactory{}, &fakeMyParcelClientFactory{}, testEncSecret, "https://api.example.com", &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "conn-1", "order.updated")

	require.ErrorIs(t, err, integrationsDomain.ErrWebhookSecretNotSet)
}

func TestDeleteProviderWebhookUseCase_woo_deletesByID(t *testing.T) {
	conn := wooConn(nil)
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	var gotID string
	wooClient := &fakeFullWooClient{deleteWebhookFn: func(_ context.Context, id string) error { gotID = id; return nil }}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return wooClient, nil }}
	uc := appintegrations.NewDeleteProviderWebhookUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{}, testEncSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "conn-1", "42")

	require.NoError(t, err)
	assert.Equal(t, "42", gotID)
}

func TestDeleteProviderWebhookUseCase_myParcel_deletesByID(t *testing.T) {
	conn := myParcelConn(nil)
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	var gotID string
	client := &fakeFullMyParcelClient{deleteWebhookSubscriptionFn: func(_ context.Context, id string) error { gotID = id; return nil }}
	uc := appintegrations.NewDeleteProviderWebhookUseCase(newMockGormDB(t), connRepo, &mocks.WooClientFactory{}, &fakeMyParcelClientFactory{client: client}, testEncSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "conn-1", "9")

	require.NoError(t, err)
	assert.Equal(t, "9", gotID)
}

func TestDeleteProviderWebhookUseCase_liveCallFailure_wrapsErrProviderWebhookRequestFailed(t *testing.T) {
	conn := wooConn(nil)
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	wooClient := &fakeFullWooClient{deleteWebhookFn: func(_ context.Context, id string) error { return errors.New("not found") }}
	wooFactory := &mocks.WooClientFactory{NewClientFn: func(_, _, _ string) (integrationsDomain.WooClient, error) { return wooClient, nil }}
	uc := appintegrations.NewDeleteProviderWebhookUseCase(newMockGormDB(t), connRepo, wooFactory, &fakeMyParcelClientFactory{}, testEncSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "conn-1", "42")

	require.ErrorIs(t, err, integrationsDomain.ErrProviderWebhookRequestFailed)
}
