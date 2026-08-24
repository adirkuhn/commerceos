//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// fakeBolPusher stands in for the real PushProductToBolUseCase -- this test
// isolates RunScheduledBolExportUseCase's enrollment-resolution logic
// (SalesChannel + override + product, all real repos/real Postgres) from
// the actual Bol API call, which is a separate, already-tested concern.
type fakeBolPusher struct {
	calls []appchannels.PushProductToBolInput
}

func (f *fakeBolPusher) Execute(_ context.Context, in appchannels.PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error) {
	f.calls = append(f.calls, in)
	return []channelsDomain.ProductChannelListing{{ID: "listing-1", ProductID: in.ProductID, ConnectionID: in.ConnectionID}}, nil
}

// TestRunScheduledBolExportUseCase_Integration_PushesAManuallyEnrolledProduct
// reproduces a real prod scenario byte-for-byte: a Bol connection, a
// SalesChannel (type=bol, enrollment_mode=manual, active=true) pointed at
// it, a single-SKU product with a real barcode, and a
// SalesChannelProductOverride(included=true) for that exact product+channel
// pair -- exactly what a real "Add" click produces. Prod showed this
// schedule reporting status=success with zero effect (no
// product_channel_listings row, no channel_sync_logs row) despite this
// override existing; every involved use case is otherwise fully unit-tested
// with sqlmock, so if there's a real bug here it's specifically in how the
// real repos' queries interact -- something sqlmock structurally can't
// catch (same class as the CHECK-constraint bugs found earlier).
func TestRunScheduledBolExportUseCase_Integration_PushesAManuallyEnrolledProduct(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	connRepo := repositories.NewConnectionRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	channelRepo := repositories.NewSalesChannelRepo(db)
	overrideRepo := repositories.NewSalesChannelProductOverrideRepo(db)

	var conn integrationsDomain.Connection
	var channel channelsDomain.SalesChannel
	var product pmsDomain.Product
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()

		conn = integrationsDomain.Connection{
			TenantID: tenant.IDFromContext(ctx), Provider: integrationsDomain.ProviderBol,
			ConsumerKey: "client-id", ConsumerSecretEncrypted: "encrypted-secret",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Create(ctx, &conn); err != nil {
			return err
		}

		channel = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "BOL", Type: channelsDomain.ChannelTypeBol,
			ConnectionID: &conn.ID, EnrollmentMode: channelsDomain.EnrollmentModeManual,
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := channelRepo.Create(ctx, &channel); err != nil {
			return err
		}

		product = pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Creme - Transparente - Hipoglos - 30g",
			Slug: "creme-transparente-hipoglos-30g", Status: pmsDomain.ProductStatusActive,
			HasVariants: false, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, &product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "HIPO-30G",
			Barcode: "7891010249939", PriceMinorUnits: 1699, CurrencyCode: "EUR",
			Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}

		override := &channelsDomain.SalesChannelProductOverride{
			TenantID: tenant.IDFromContext(ctx), ChannelID: channel.ID, ProductID: product.ID,
			Included: true, CreatedAt: now,
		}
		return overrideRepo.Upsert(ctx, override)
	})
	require.NoError(t, err)

	pusher := &fakeBolPusher{}
	uc := appchannels.NewRunScheduledBolExportUseCase(db, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(ctx, appchannels.RunScheduledBolExportInput{ConnectionID: conn.ID})

	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Equal(t, 1, result.Pushed, "the manually-enrolled product must be pushed")
	require.Len(t, pusher.calls, 1)
	require.Equal(t, product.ID, pusher.calls[0].ProductID)
	require.Equal(t, conn.ID, pusher.calls[0].ConnectionID)
}
