//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func seedMyParcelConnection(t *testing.T, db *gorm.DB, ctx context.Context) string {
	t.Helper()
	connRepo := repositories.NewConnectionRepo(db)
	var connectionID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn := &integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderMyParcel,
			ConsumerSecretEncrypted: "enc", Status: integrationsDomain.StatusConnected,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, conn); err != nil {
			return err
		}
		connectionID = conn.ID
		return nil
	})
	require.NoError(t, err)
	return connectionID
}

func TestShipmentRepo_Integration_CreateFindByIDAndByMyParcelID(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	connectionID := seedMyParcelConnection(t, db, ctx)
	repo := repositories.NewShipmentRepo(db)

	var created shippingDomain.Shipment
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = shippingDomain.Shipment{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, ConnectionID: connectionID,
			Carrier: shippingDomain.CarrierPostNL, MyParcelShipmentID: 999, Barcode: "3SMYPA1",
			Status: 1, StatusMain: "registered", LabelFormat: shippingDomain.LabelFormatPDF,
			PackageType:     shippingDomain.PackageTypePackage,
			RecipientPerson: "Jane Doe", RecipientStreet: "Damrak", RecipientNumber: "1",
			RecipientPostalCode: "1012AB", RecipientCity: "Amsterdam", RecipientCC: "NL",
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var byID *shippingDomain.Shipment
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byID, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "3SMYPA1", byID.Barcode)
	require.Equal(t, shippingDomain.CarrierPostNL, byID.Carrier)

	var byMPID *shippingDomain.Shipment
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byMPID, err = repo.FindByMyParcelShipmentID(ctx, 999)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, byMPID.ID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByMyParcelShipmentID(ctx, 424242)
		return err
	})
	require.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound)
}

func TestShipmentRepo_Integration_UpdateStatusAndFindStaleNonTerminal(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	connectionID := seedMyParcelConnection(t, db, ctx)
	repo := repositories.NewShipmentRepo(db)

	var shipmentID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		s := shippingDomain.Shipment{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, ConnectionID: connectionID,
			Carrier: shippingDomain.CarrierPostNL, MyParcelShipmentID: 1000, Barcode: "3SMYPA2",
			Status: 1, StatusMain: "registered", LabelFormat: shippingDomain.LabelFormatPDF,
			RecipientPerson: "P", RecipientStreet: "S", RecipientCity: "C", RecipientCC: "NL",
		}
		if err := repo.Create(ctx, &s); err != nil {
			return err
		}
		shipmentID = s.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.UpdateStatus(ctx, shipmentID, 2, "shipped")
	})
	require.NoError(t, err)

	var found *shippingDomain.Shipment
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, shipmentID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "shipped", found.StatusMain)

	// A shipment updated "just now" is not stale against a very old cutoff.
	var stale []shippingDomain.Shipment
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		stale, err = repo.FindStaleNonTerminal(ctx, time.Now().Add(-time.Hour), 0)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, stale)

	// Against a future cutoff, the non-terminal ("shipped") shipment is stale.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		stale, err = repo.FindStaleNonTerminal(ctx, time.Now().Add(time.Hour), 0)
		return err
	})
	require.NoError(t, err)
	require.Len(t, stale, 1)
	require.Equal(t, shipmentID, stale[0].ID)

	// Mark it delivered (terminal) -- must drop out of the stale scan even
	// against a far-future cutoff.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.UpdateStatus(ctx, shipmentID, 99, "delivered")
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		stale, err = repo.FindStaleNonTerminal(ctx, time.Now().Add(time.Hour), 0)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, stale, "a delivered (terminal) shipment must never be re-polled")
}

func TestShipmentStatusEventRepo_Integration_Append(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	orderID := seedSalesOrderForPickingDB(t, db, ctx, warehouseID)
	connectionID := seedMyParcelConnection(t, db, ctx)
	shipmentRepo := repositories.NewShipmentRepo(db)
	eventRepo := repositories.NewShipmentStatusEventRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		s := shippingDomain.Shipment{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, ConnectionID: connectionID,
			Carrier: shippingDomain.CarrierPostNL, MyParcelShipmentID: 2000, Barcode: "3SMYPA3",
			LabelFormat:     shippingDomain.LabelFormatPDF,
			RecipientPerson: "P", RecipientStreet: "S", RecipientCity: "C", RecipientCC: "NL",
		}
		if err := shipmentRepo.Create(ctx, &s); err != nil {
			return err
		}
		event := shippingDomain.ShipmentStatusEvent{
			TenantID: tenant.IDFromContext(ctx), ShipmentID: s.ID, Status: 2, StatusMain: "shipped",
			Source: shippingDomain.StatusEventSourceWebhook, OccurredAt: time.Now(),
		}
		return eventRepo.Append(ctx, &event)
	})
	require.NoError(t, err)
}

func TestShipmentRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, superDB, ctxA)
	orderID := seedSalesOrderForPickingDB(t, superDB, ctxA, warehouseID)
	connectionID := seedMyParcelConnection(t, superDB, ctxA)
	repo := repositories.NewShipmentRepo(db)

	var shipmentID string
	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		s := shippingDomain.Shipment{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, ConnectionID: connectionID,
			Carrier: shippingDomain.CarrierPostNL, MyParcelShipmentID: 3000, Barcode: "3SMYPA4",
			LabelFormat:     shippingDomain.LabelFormatPDF,
			RecipientPerson: "P", RecipientStreet: "S", RecipientCity: "C", RecipientCC: "NL",
		}
		if err := repo.Create(ctx, &s); err != nil {
			return err
		}
		shipmentID = s.ID
		return nil
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, shipmentID)
		return err
	})
	require.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound, "tenant B must not see tenant A's shipment")
}

func TestShippingConnectionSettingsRepo_Integration_UpsertAndFind(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connectionID := seedMyParcelConnection(t, db, ctx)
	repo := repositories.NewShippingConnectionSettingsRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		s := &shippingDomain.ShippingConnectionSettings{
			ConnectionID: connectionID, TenantID: tenant.IDFromContext(ctx),
			DefaultCarrier: shippingDomain.CarrierPostNL, DefaultPackageType: shippingDomain.PackageTypePackage,
			DefaultLabelFormat: shippingDomain.LabelFormatPDF, DefaultSignatureRequired: false,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		return repo.Upsert(ctx, s)
	})
	require.NoError(t, err)

	var found *shippingDomain.ShippingConnectionSettings
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByConnectionID(ctx, connectionID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, shippingDomain.CarrierPostNL, found.DefaultCarrier)

	// Upsert again (a settings edit) must update in place, not duplicate.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		s := &shippingDomain.ShippingConnectionSettings{
			ConnectionID: connectionID, TenantID: tenant.IDFromContext(ctx),
			DefaultCarrier: shippingDomain.CarrierDPD, DefaultPackageType: shippingDomain.PackageTypePackage,
			DefaultLabelFormat: shippingDomain.LabelFormatZPL, DefaultSignatureRequired: true,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		return repo.Upsert(ctx, s)
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByConnectionID(ctx, connectionID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, shippingDomain.CarrierDPD, found.DefaultCarrier)
	require.True(t, found.DefaultSignatureRequired)
}

func TestShippingConnectionSettingsRepo_Integration_NotFoundReturnsNilNotError(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewShippingConnectionSettingsRepo(db)

	var found *shippingDomain.ShippingConnectionSettings
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByConnectionID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestShippingCarrierOverrideRepo_Integration_UpsertListAndFind(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	connectionID := seedMyParcelConnection(t, db, ctx)
	settingsRepo := repositories.NewShippingConnectionSettingsRepo(db)
	repo := repositories.NewShippingCarrierOverrideRepo(db)

	// A carrier override references shipping_connection_settings(connection_id)
	// (migration 096) -- the global settings row must exist first.
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return settingsRepo.Upsert(ctx, &shippingDomain.ShippingConnectionSettings{
			ConnectionID: connectionID, TenantID: tenant.IDFromContext(ctx),
			DefaultCarrier: shippingDomain.CarrierPostNL, DefaultPackageType: shippingDomain.PackageTypePackage,
			DefaultLabelFormat: shippingDomain.LabelFormatPDF,
			CreatedAt:          time.Now(), UpdatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	pkgType := shippingDomain.PackageTypeMailboxPackage
	sig := true
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		o := &shippingDomain.ShippingCarrierOverride{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: connectionID, Carrier: shippingDomain.CarrierPostNL,
			PackageType: &pkgType, SignatureRequired: &sig,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		return repo.Upsert(ctx, o)
	})
	require.NoError(t, err)

	var list []shippingDomain.ShippingCarrierOverride
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		list, err = repo.ListByConnection(ctx, connectionID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, shippingDomain.PackageTypeMailboxPackage, *list[0].PackageType)

	var found *shippingDomain.ShippingCarrierOverride
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByConnectionAndCarrier(ctx, connectionID, shippingDomain.CarrierPostNL)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.True(t, *found.SignatureRequired)

	// A carrier with no override row returns nil, not an error.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByConnectionAndCarrier(ctx, connectionID, shippingDomain.CarrierDPD)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}
