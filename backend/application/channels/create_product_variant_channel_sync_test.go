package channels_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// noopSKUPriceHistoryRepo satisfies pmsDomain.SKUPriceHistoryRepository --
// this test doesn't care about price history, only that the SKU gets
// created and the product reaches WooCommerce.
type noopSKUPriceHistoryRepo struct{}

func (noopSKUPriceHistoryRepo) Create(context.Context, *pmsDomain.SKUPriceHistory) error { return nil }
func (noopSKUPriceHistoryRepo) CloseOpen(context.Context, string, time.Time) error       { return nil }
func (noopSKUPriceHistoryRepo) ListBySKU(context.Context, string, int) ([]pmsDomain.SKUPriceHistory, error) {
	return nil, nil
}

// TestCreateProductThenAddVariant_endToEnd_reachesWooCommerce is a real,
// composed-use-case regression test (not narrowly-stubbed unit tests) for
// the exact reported bug: create a REGULAR product (HasVariants: false --
// the plain/flat product type, not a true multi-variant one), add its one
// variant and SKU (CreateVariantUseCase explicitly allows exactly one
// variant on a HasVariants:false product -- this is the "default variant"
// the original bug report meant), then run the real enrollment path
// (EnrollProductInChannelsUseCase -> PushProductToWooUseCase) and confirm
// it actually reaches WooCommerce.
//
// This is also why enrolling right after CreateProductUseCase (simulating
// pms.product.changed firing immediately, before any variant exists, same
// as ProductChangedConsumer would in production) is asserted to push
// NOTHING -- that's the original failure mode, not a bug in itself. The
// fix is that CreateVariantUseCase/CreateSKUUseCase now raise the same
// event once the variant/SKU actually exist, which is what the second
// enroll call below simulates.
func TestCreateProductThenAddVariant_endToEnd_reachesWooCommerce(t *testing.T) {
	connID := "conn-1"
	h := newHarness(nil)
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: "sku-1", Syncable: 36}}
	outboxRepo := &fakeOutboxRepo{}
	db := newMockGormDB(t, 20) // generous: product+variant+sku creates, two enroll passes, one real push

	createProductUC := apppms.NewCreateProductUseCase(db, h.productRepo, outboxRepo)
	createVariantUC := apppms.NewCreateVariantUseCase(db, h.productRepo, h.variantRepo, outboxRepo)
	createSKUUC := apppms.NewCreateSKUUseCase(db, h.variantRepo, h.skuRepo, noopSKUPriceHistoryRepo{}, outboxRepo)
	pushUC := appchannels.NewPushProductToWooUseCase(
		db, h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{},
	)
	enrollUC := appchannels.NewEnrollProductInChannelsUseCase(db, h.productRepo, channelRepo, overrideRepo, pushUC, nil)

	// 1. Create a regular (non-variant-mode) product -- no variant/SKU yet.
	product, err := createProductUC.Execute(context.Background(), apppms.CreateProductInput{
		Name: "Xarope de Guaraná", Status: pmsDomain.ProductStatusActive, HasVariants: false,
	})
	require.NoError(t, err)
	h.productRepo.byID = map[string]pmsDomain.Product{product.ID: *product}

	// Simulate ProductChangedConsumer reacting to CreateProductUseCase's own
	// pms.product.changed -- fired before any variant exists.
	require.NoError(t, enrollUC.Execute(context.Background(), product.ID))
	require.Empty(t, h.wooClient.createCalled, "nothing to push yet -- no variant/SKU exists")

	// 2. Add the product's one ("default") variant.
	variant, err := createVariantUC.Execute(context.Background(), apppms.CreateVariantInput{ProductID: product.ID, Name: "900ml"})
	require.NoError(t, err)
	h.variantRepo.findByIDResult = variant
	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{product.ID: {*variant}}

	// 3. Add that variant's SKU.
	sku, err := createSKUUC.Execute(context.Background(), apppms.CreateSKUInput{
		VariantID: variant.ID, SKUCode: "GUARANA-900", PriceMinorUnits: 999, CurrencyCode: "EUR",
		Status: pmsDomain.SKUStatusActive,
	})
	require.NoError(t, err)
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{variant.ID: {*sku}}

	// Simulate ProductChangedConsumer reacting to CreateSKUUseCase's own
	// pms.product.changed -- the fix's whole point: this must actually
	// re-trigger enrollment and reach WooCommerce this time.
	require.NoError(t, enrollUC.Execute(context.Background(), product.ID))

	require.Len(t, h.wooClient.createCalled, 1, "the default/only variant must reach WooCommerce once it exists")
	pushed := h.wooClient.createCalled[0]
	require.Equal(t, "GUARANA-900", pushed.SKU)
	require.Equal(t, "9.99", pushed.RegularPrice)
	require.Len(t, h.listingRepo.created, 1, "a ProductChannelListing must exist so the UI shows this product as synced")
}
