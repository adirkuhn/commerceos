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
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", ExternalShipmentID: "555", LabelFormat: shippingDomain.LabelFormatZPL}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil }}

	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
	}

	var gotID string
	var gotFormat, gotSize string
	client := &fakeCarrierClient{getLabelFn: func(_ context.Context, id string, format, size string) (integrationsDomain.LabelResult, error) {
		gotID, gotFormat, gotSize = id, format, size
		return integrationsDomain.LabelResult{Data: []byte("^XA^XZ"), ContentType: "application/vnd.shipment_label+zpl"}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeCarrierClientFactory{client: client}, testEncSecret, &fakeShipmentLabelRepo{})

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "")

	require.NoError(t, err)
	assert.Equal(t, "555", gotID)
	assert.Equal(t, "zpl", gotFormat)
	assert.Equal(t, "A4", gotSize, "falls back to LabelSizeA4 when no size requested and no connection default configured")
	assert.Equal(t, []byte("^XA^XZ"), result.Data)
}

// TestGetShipmentLabel_explicitSizeWinsOverConnectionDefault proves a
// per-print size request (the A6 print-time option) overrides the
// connection's own DefaultLabelSize, not the other way around.
func TestGetShipmentLabel_explicitSizeWinsOverConnectionDefault(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", ExternalShipmentID: "555", LabelFormat: shippingDomain.LabelFormatPDF}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultLabelSize: "A4"}, nil
		},
	}
	var gotSize string
	client := &fakeCarrierClient{getLabelFn: func(_ context.Context, _, _, size string) (integrationsDomain.LabelResult, error) {
		gotSize = size
		return integrationsDomain.LabelResult{}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeCarrierClientFactory{client: client}, testEncSecret, &fakeShipmentLabelRepo{})
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "A6")

	require.NoError(t, err)
	assert.Equal(t, "A6", gotSize)
}

// TestGetShipmentLabel_noExplicitSizeFallsBackToConnectionDefault proves an
// empty size request falls back to the connection's own configured default
// (the "configurable" part of the A6 feature), not always A4.
func TestGetShipmentLabel_noExplicitSizeFallsBackToConnectionDefault(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", ExternalShipmentID: "555", LabelFormat: shippingDomain.LabelFormatPDF}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultLabelSize: "A6"}, nil
		},
	}
	var gotSize string
	client := &fakeCarrierClient{getLabelFn: func(_ context.Context, _, _, size string) (integrationsDomain.LabelResult, error) {
		gotSize = size
		return integrationsDomain.LabelResult{}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeCarrierClientFactory{client: client}, testEncSecret, &fakeShipmentLabelRepo{})
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "")

	require.NoError(t, err)
	assert.Equal(t, "A6", gotSize)
}

func TestGetShipmentLabel_shipmentNotFound_propagatesError(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) {
		return nil, shippingDomain.ErrShipmentNotFound
	}}
	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, &mocks.ConnectionRepo{}, &mocks.ShippingConnectionSettingsRepo{}, &fakeCarrierClientFactory{}, testEncSecret, &fakeShipmentLabelRepo{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "missing", "")

	require.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound)
}

func TestGetShipmentLabel_storedLabelIsServedWithoutAskingTheCarrier(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", ExternalShipmentID: "ref", LabelFormat: shippingDomain.LabelFormatPDF}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(context.Context, string) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderPostNL, ConsumerSecretEncrypted: encryptedFor(t, "k")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(context.Context, string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
	}
	labels := &fakeShipmentLabelRepo{stored: &shippingDomain.ShipmentLabel{ShipmentID: "s-1", ContentType: "application/pdf", Data: []byte("%PDF stored")}}
	client := &fakeCarrierClient{getLabelFn: func(context.Context, string, string, string) (integrationsDomain.LabelResult, error) {
		t.Fatal("the carrier must not be asked for a label that is stored")
		return integrationsDomain.LabelResult{}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, settingsRepo, &fakeCarrierClientFactory{client: client}, testEncSecret, labels)
	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1", "A6")

	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF stored"), result.Data)
	assert.Equal(t, "application/pdf", result.ContentType)
}
