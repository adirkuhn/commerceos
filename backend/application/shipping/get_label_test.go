package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestGetShipmentLabel_fetchesInShipmentsOwnFormat(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", MyParcelShipmentID: 555, LabelFormat: shippingDomain.LabelFormatZPL}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil }}

	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
	}

	var gotID int64
	var gotFormat, gotSize string
	client := &fakeMyParcelClient{getLabelFn: func(_ context.Context, id int64, format, size string) (integrationsDomain.LabelResult, error) {
		gotID, gotFormat, gotSize = id, format, size
		return integrationsDomain.LabelResult{Data: []byte("^XA^XZ"), ContentType: "application/vnd.shipment_label+zpl"}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeMyParcelClientFactory{client: client}, testEncSecret)

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "")

	require.NoError(t, err)
	assert.Equal(t, int64(555), gotID)
	assert.Equal(t, "zpl", gotFormat)
	assert.Equal(t, "A4", gotSize, "falls back to LabelSizeA4 when no size requested and no connection default configured")
	assert.Equal(t, []byte("^XA^XZ"), result.Data)
}

// TestGetShipmentLabel_explicitSizeWinsOverConnectionDefault proves a
// per-print size request (the A6 print-time option) overrides the
// connection's own DefaultLabelSize, not the other way around.
func TestGetShipmentLabel_explicitSizeWinsOverConnectionDefault(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", MyParcelShipmentID: 555, LabelFormat: shippingDomain.LabelFormatPDF}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultLabelSize: "A4"}, nil
		},
	}
	var gotSize string
	client := &fakeMyParcelClient{getLabelFn: func(_ context.Context, _ int64, _, size string) (integrationsDomain.LabelResult, error) {
		gotSize = size
		return integrationsDomain.LabelResult{}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeMyParcelClientFactory{client: client}, testEncSecret)
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "A6")

	require.NoError(t, err)
	assert.Equal(t, "A6", gotSize)
}

// TestGetShipmentLabel_noExplicitSizeFallsBackToConnectionDefault proves an
// empty size request falls back to the connection's own configured default
// (the "configurable" part of the A6 feature), not always A4.
func TestGetShipmentLabel_noExplicitSizeFallsBackToConnectionDefault(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", MyParcelShipmentID: 555, LabelFormat: shippingDomain.LabelFormatPDF}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultLabelSize: "A6"}, nil
		},
	}
	var gotSize string
	client := &fakeMyParcelClient{getLabelFn: func(_ context.Context, _ int64, _, size string) (integrationsDomain.LabelResult, error) {
		gotSize = size
		return integrationsDomain.LabelResult{}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeMyParcelClientFactory{client: client}, testEncSecret)
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "")

	require.NoError(t, err)
	assert.Equal(t, "A6", gotSize)
}

func TestGetShipmentLabel_shipmentNotFound_propagatesError(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) {
		return nil, shippingDomain.ErrShipmentNotFound
	}}
	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, &mocks.ConnectionRepo{}, &mocks.ShippingConnectionSettingsRepo{}, &fakeMyParcelClientFactory{}, testEncSecret)

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "missing", "")

	require.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound)
}
