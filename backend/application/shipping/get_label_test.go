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

	var gotID int64
	var gotFormat string
	client := &fakeMyParcelClient{getLabelFn: func(_ context.Context, id int64, format string) (integrationsDomain.LabelResult, error) {
		gotID, gotFormat = id, format
		return integrationsDomain.LabelResult{Data: []byte("^XA^XZ"), ContentType: "application/vnd.shipment_label+zpl"}, nil
	}}

	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, connRepo, &fakeMyParcelClientFactory{client: client}, testEncSecret)

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "s-1")

	require.NoError(t, err)
	assert.Equal(t, int64(555), gotID)
	assert.Equal(t, "zpl", gotFormat)
	assert.Equal(t, []byte("^XA^XZ"), result.Data)
}

func TestGetShipmentLabel_shipmentNotFound_propagatesError(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) {
		return nil, shippingDomain.ErrShipmentNotFound
	}}
	uc := appshipping.NewGetShipmentLabelUseCase(newMockGormDB(t, 1), shipmentRepo, &mocks.ConnectionRepo{}, &fakeMyParcelClientFactory{}, testEncSecret)

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "missing")

	require.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound)
}
