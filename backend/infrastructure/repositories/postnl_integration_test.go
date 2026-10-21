//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// The PostNL columns need explicit gorm column tags (the default namer
// would look for "post_nl_customer_code") -- only a real schema catches a
// mismatch.
func TestConnectionRepo_Integration_RoundTripsPostNLFields(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewConnectionRepo(db)
	conn := &integrationsDomain.Connection{
		Provider: integrationsDomain.ProviderPostNL, ConsumerSecretEncrypted: "enc", Status: integrationsDomain.StatusConnected,
		PostNLCustomerCode: "DEVC", PostNLCustomerNumber: "11223344", PostNLSandbox: true,
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		conn.TenantID = tenant.IDFromContext(ctx)
		conn.CreatedAt, conn.UpdatedAt = time.Now(), time.Now()
		return repo.Create(ctx, conn)
	}))

	var found *integrationsDomain.Connection
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, conn.ID)
		return err
	}))
	assert.Equal(t, "DEVC", found.PostNLCustomerCode)
	assert.Equal(t, "11223344", found.PostNLCustomerNumber)
	assert.True(t, found.PostNLSandbox)

	found.PostNLCustomerNumber, found.PostNLSandbox = "99887766", false
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error { return repo.Update(ctx, found) }))
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, conn.ID)
		return err
	}))
	assert.Equal(t, "99887766", found.PostNLCustomerNumber)
	assert.False(t, found.PostNLSandbox)
}

func TestShipmentLabelRepo_Integration_SaveAndFind(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	connectionID := seedMyParcelConnection(t, db, ctx)
	shipments := repositories.NewShipmentRepo(db)
	labels := repositories.NewShipmentLabelRepo(db)

	var shipment shippingDomain.Shipment
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		shipment = shippingDomain.Shipment{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, ConnectionID: connectionID,
			Carrier: shippingDomain.CarrierPostNL, ExternalShipmentID: "ref-1", Barcode: "3SDEVC1",
			StatusMain: "registered", LabelFormat: shippingDomain.LabelFormatPDF, PackageType: shippingDomain.PackageTypePackage,
			RecipientPerson: "Jane Doe", RecipientStreet: "Damrak", RecipientNumber: "1",
			RecipientPostalCode: "1012AB", RecipientCity: "Amsterdam", RecipientCC: "NL",
		}
		if err := shipments.Create(ctx, &shipment); err != nil {
			return err
		}
		return labels.Save(ctx, &shippingDomain.ShipmentLabel{
			ShipmentID: shipment.ID, TenantID: tenant.IDFromContext(ctx), ContentType: "application/pdf",
			Data: []byte("%PDF-1.4 label"), CreatedAt: time.Now(),
		})
	}))

	var found, missing *shippingDomain.ShipmentLabel
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if found, err = labels.FindByShipmentID(ctx, shipment.ID); err != nil {
			return err
		}
		missing, err = labels.FindByShipmentID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	}))
	require.NotNil(t, found)
	assert.Equal(t, "application/pdf", found.ContentType)
	assert.Equal(t, []byte("%PDF-1.4 label"), found.Data)
	assert.Nil(t, missing)
}
