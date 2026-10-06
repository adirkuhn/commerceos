package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// stubBolOfferGater backs bolOfferGater (the narrow SetOnHold-only
// interface SyncProductToChannelsUseCase uses to gate/ungate a Bol
// listing) -- distinct from stubBolProductPusher, which is the full offer
// push interface used for everything else Bol.
type stubBolOfferGater struct {
	calls []stubOnHoldCall
	err   error
}

type stubOnHoldCall struct {
	ConnectionID string
	ExternalID   string
	OnHold       bool
}

func (s *stubBolOfferGater) SetOnHold(_ context.Context, connectionID, externalID string, onHold bool) error {
	s.calls = append(s.calls, stubOnHoldCall{ConnectionID: connectionID, ExternalID: externalID, OnHold: onHold})
	return s.err
}

// stubStockForceSyncer backs productStockForceSyncer -- the stock catch-up
// dependency SyncProductToChannelsUseCase calls once per SKU when a
// connection ungates.
type stubStockForceSyncer struct {
	calls []string
	err   error
}

func (s *stubStockForceSyncer) ForceExecute(_ context.Context, skuID string) error {
	s.calls = append(s.calls, skuID)
	return s.err
}

func newSyncHarness(product pmsDomain.Product, variant pmsDomain.Variant, sku pmsDomain.SKU, listings []channelsDomain.ProductChannelListing) *harness {
	h := newPushHarness(product, variant, sku)
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{product.ID: listings}
	return h
}

func (h *harness) newSyncUseCase(t *testing.T, n int) *appchannels.SyncProductToChannelsUseCase {
	t.Helper()
	return h.newSyncUseCaseWithPusher(t, n, &stubProductPusher{})
}

func (h *harness) newSyncUseCaseWithPusher(t *testing.T, n int, pusher *stubProductPusher) *appchannels.SyncProductToChannelsUseCase {
	t.Helper()
	return appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, n), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, pusher, h.mediaRepo, &stubCategoryLinkResolver{}, nil, nil, nil, &eventrecordtest.Outbox{},
	)
}

func TestSyncProductToChannels_noListings_isNoOp(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newSyncHarness(product, variant, sku, nil)
	uc := h.newSyncUseCase(t, 1)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled)
}

func TestSyncProductToChannels_pushesUpdatedFieldsToWoo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	call := h.wooClient.updateProductCalled[0]
	require.Equal(t, "555", call.ExternalID)
	require.Equal(t, product.Name, call.Product.Name)
	require.Equal(t, product.Description, call.Product.Description)
	require.Equal(t, "publish", call.Product.Status)
	require.Equal(t, "19.99", call.Product.RegularPrice)

	require.Len(t, h.listingRepo.updated, 1)
	require.Equal(t, channelsDomain.SyncStatusSynced, h.listingRepo.updated[0].SyncStatus)
	require.NotNil(t, h.listingRepo.updated[0].LastSyncedAt)
}

// TestSyncProductToChannels_pushesUpdatedImageGalleryToWoo proves an edit
// to a product's media (upload/reorder/delete) reaches Woo the same way
// every other product-changed edit does -- through this use case's
// pms.product.changed-triggered re-push, not a separate image-sync path.
func TestSyncProductToChannels_pushesUpdatedImageGalleryToWoo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.mediaRepo.byProduct = map[string][]pmsDomain.Media{
		product.ID: {{ID: "media-1", ProductID: &product.ID, URL: "https://cdn.example.com/front.jpg", AltText: "Front", Position: 0}},
	}
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	require.Equal(t, []integrationsDomain.WooProductImage{{URL: "https://cdn.example.com/front.jpg", Alt: "Front"}},
		h.wooClient.updateProductCalled[0].Product.Images)
}

// TestSyncProductToChannels_pushesUpdatedSKUToWoo proves an edit to a
// product's SKU code reaches Woo on re-sync -- previously buildWooProduct
// never set WooProduct.SKU on the update path, so a SKU change made after
// initial push never propagated.
func TestSyncProductToChannels_pushesUpdatedSKUToWoo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.SKUCode = "TSHIRT-1-NEW"
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	require.Equal(t, "TSHIRT-1-NEW", h.wooClient.updateProductCalled[0].Product.SKU)
}

func TestSyncProductToChannels_pushesUpdatedBarcodeAsGTIN(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.Barcode = "0012345678905"
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	require.Equal(t, "0012345678905", h.wooClient.updateProductCalled[0].Product.GTIN)
}

// TestSyncProductToChannels_setsWooCategoryWhenLinkResolves proves a
// product's category is resolved and pushed on every re-sync, not just the
// initial push -- the same categoryLinkResolver PushProductToWooUseCase
// uses.
func TestSyncProductToChannels_setsWooCategoryWhenLinkResolves(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	categoryID := "cat-1"
	product.CategoryID = &categoryID
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 4), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{externalID: "20"}, nil, nil, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	require.Equal(t, []integrationsDomain.WooProductCategory{{ExternalID: "20"}}, h.wooClient.updateProductCalled[0].Product.Categories)
}

// TestSyncProductToChannels_syncsSalePriceExpiry proves the actual path
// the "sync sale-price expiry to Woo" requirement depends on: this is the
// exact event (pms.product.changed) ExpireSKUSalePriceUseCase raises after
// clearing a SKU's sale, and this is the exact consumer/use case that
// picks it up and pushes the reverted price to Woo -- no separate Woo-sync
// mechanism was built for this, it reuses the one every other product edit
// already goes through.
func TestSyncProductToChannels_syncsSalePriceExpiry(t *testing.T) {
	product, variant, sku := simplePhysicalProduct() // no sale price set -- simulates post-expiry state
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	call := h.wooClient.updateProductCalled[0]
	require.Equal(t, "19.99", call.Product.RegularPrice, "regular price still pushed")
	require.Empty(t, call.Product.SalePrice, "no sale price sent once expired")
}

func TestSyncProductToChannels_convertsWeightAndDimensionsUsingStoreUnits(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	weight, length, width, height := int64(500), int64(300), int64(200), int64(100) // grams/mm
	sku.WeightGrams = &weight
	sku.LengthMM = &length
	sku.WidthMM = &width
	sku.HeightMM = &height
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.weightUnit = "kg"
	h.wooClient.dimensionUnit = "cm"
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	call := h.wooClient.updateProductCalled[0].Product
	require.Equal(t, "0.5", call.Weight)
	require.Equal(t, "30", call.Dimensions.Length)
	require.Equal(t, "20", call.Dimensions.Width)
	require.Equal(t, "10", call.Dimensions.Height)
}

func TestSyncProductToChannels_neverSetsWeightOrDimensionsForVirtualProducts(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.IsVirtual = true
	weight := int64(500)
	sku.WeightGrams = &weight
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.weightUnit = "kg"
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled[0].Product.Weight)
	require.True(t, h.wooClient.updateProductCalled[0].Product.Virtual)
}

func TestSyncProductToChannels_recordsErrorStatusWhenWooUpdateFails_withoutFailingTheUseCase(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.updateProductErr = errors.New("woo unreachable")
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.listingRepo.updated, 1)
	require.Equal(t, channelsDomain.SyncStatusError, h.listingRepo.updated[0].SyncStatus)
	require.Contains(t, h.listingRepo.updated[0].LastSyncError, "woo unreachable")
}

// TestSyncProductToChannels_migratesShapeWhenProductHasGrownVariants covers
// a product pushed while single-SKU, then given variants afterwards: its
// one legacy listing row (VariantID nil) no longer matches any shape
// HasVariants=true calls for, so the old Woo product is torn down and the
// product is re-pushed from scratch via PushProductToWooUseCase in
// whichever shape SyncVariantsAsWooVariations now calls for.
func TestSyncProductToChannels_migratesShapeWhenProductHasGrownVariants(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.HasVariants = true // legacy single-SKU listing is now out of shape
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	pusher := &stubProductPusher{}
	uc := h.newSyncUseCaseWithPusher(t, 3, pusher)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled, "no in-place update -- shape mismatch means tear down and re-push")
	require.Equal(t, []string{"555"}, h.wooClient.deleteProductCalled)
	require.Equal(t, []string{"listing-1"}, h.listingRepo.deleted)
	require.Len(t, pusher.calls, 1)
	require.Equal(t, product.ID, pusher.calls[0].ProductID)
	require.Equal(t, "conn-1", pusher.calls[0].ConnectionID)
}

// twoVariantProduct is the fixture for every variant-sync-mode test below —
// a HasVariants=true product with two Variant/SKU pairs, e.g. two
// pack-size variants of the same underlying item.
func twoVariantProduct(syncAsVariations bool) (pmsDomain.Product, []pmsDomain.Variant, []pmsDomain.SKU) {
	product := pmsDomain.Product{
		ID: "product-1", Name: "Carne Seca", Slug: "carne-seca", Description: "Dried salted meat",
		Status: pmsDomain.ProductStatusActive, HasVariants: true, SyncVariantsAsWooVariations: syncAsVariations,
	}
	variants := []pmsDomain.Variant{
		{ID: "variant-1", ProductID: product.ID, Name: "378g"},
		{ID: "variant-2", ProductID: product.ID, Name: "412g"},
	}
	skus := []pmsDomain.SKU{
		{ID: "sku-1", VariantID: "variant-1", SKUCode: "CARNE-378", PriceMinorUnits: 2999, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive},
		{ID: "sku-2", VariantID: "variant-2", SKUCode: "CARNE-412", PriceMinorUnits: 3299, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive},
	}
	return product, variants, skus
}

func newMultiVariantSyncHarness(product pmsDomain.Product, variants []pmsDomain.Variant, skus []pmsDomain.SKU, listings []channelsDomain.ProductChannelListing) *harness {
	h := newHarness(nil)
	h.productRepo.byID = map[string]pmsDomain.Product{product.ID: product}
	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{product.ID: variants}
	byVariant := map[string][]pmsDomain.SKU{}
	for _, sku := range skus {
		byVariant[sku.VariantID] = []pmsDomain.SKU{sku}
	}
	h.skuRepo.byVariant = byVariant
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{product.ID: listings}
	return h
}

// TestSyncProductToChannels_migratesFromSeparateProductsToVariationsWhenFlagTurnsOn
// is the "commerceos 1 product -> N variants -> woo N products" ->
// "1 product -> N variants -> woo 1 product -> N variations" case: both
// separate simple products are deleted from Woo, both old listing rows are
// deleted, and the product is re-pushed as a variable product.
func TestSyncProductToChannels_migratesFromSeparateProductsToVariationsWhenFlagTurnsOn(t *testing.T) {
	product, variants, skus := twoVariantProduct(true) // flag just turned on
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "701", VariantID: strPtr(variants[0].ID)},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "702", VariantID: strPtr(variants[1].ID)},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	pusher := &stubProductPusher{}
	uc := h.newSyncUseCaseWithPusher(t, 3, pusher)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"701", "702"}, h.wooClient.deleteProductCalled)
	require.Empty(t, h.wooClient.deleteVariationCalled)
	require.ElementsMatch(t, []string{"listing-1", "listing-2"}, h.listingRepo.deleted)
	require.Len(t, pusher.calls, 1)
	require.Equal(t, product.ID, pusher.calls[0].ProductID)
}

// TestSyncProductToChannels_migratesFromVariationsToSeparateProductsWhenFlagTurnsOff
// is the reverse toggle: the parent variable product and every one of its
// variations are deleted from Woo ("delete all other variants"), all three
// old listing rows are deleted, and the product is re-pushed as N separate
// simple products.
func TestSyncProductToChannels_migratesFromVariationsToSeparateProductsWhenFlagTurnsOff(t *testing.T) {
	product, variants, skus := twoVariantProduct(false) // flag just turned off
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "902", VariantID: strPtr(variants[1].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	pusher := &stubProductPusher{}
	uc := h.newSyncUseCaseWithPusher(t, 3, pusher)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.deleteVariationCalled, 2)
	for _, c := range h.wooClient.deleteVariationCalled {
		require.Equal(t, "900", c.ParentExternalID)
	}
	require.Equal(t, []string{"900"}, h.wooClient.deleteProductCalled)
	require.ElementsMatch(t, []string{"listing-parent", "listing-1", "listing-2"}, h.listingRepo.deleted)
	require.Len(t, pusher.calls, 1)
	require.Equal(t, product.ID, pusher.calls[0].ProductID)
}

// TestSyncProductToChannels_reTogglingBackMigratesAgain proves toggling
// twice isn't a no-op the second time either -- flipping back off after an
// earlier on (or vice versa) still migrates, since actual shape is judged
// from the listing rows currently on file, not from any memory of a prior
// state.
func TestSyncProductToChannels_reTogglingBackMigratesAgain(t *testing.T) {
	product, variants, skus := twoVariantProduct(false) // toggled back off
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "902", VariantID: strPtr(variants[1].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	pusher := &stubProductPusher{}
	uc := h.newSyncUseCaseWithPusher(t, 3, pusher)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, pusher.calls, 1, "still migrates even though this is the second flip")
	require.Equal(t, []string{"900"}, h.wooClient.deleteProductCalled)
}

// TestSyncProductToChannels_updatesExistingVariationsInPlaceWhenShapeUnchanged
// is the steady state for variations mode: shape already matches, so every
// existing variation (and the parent) is updated in place -- no
// delete/recreate churn on every ordinary product edit.
func TestSyncProductToChannels_updatesExistingVariationsInPlaceWhenShapeUnchanged(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "902", VariantID: strPtr(variants[1].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	uc := h.newSyncUseCase(t, 6)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1, "parent updated too")
	require.Equal(t, "900", h.wooClient.updateProductCalled[0].ExternalID)
	require.Len(t, h.wooClient.updateVariationCalled, 2)
	require.Empty(t, h.wooClient.createVariationCalled)
	require.Empty(t, h.wooClient.deleteVariationCalled)
	require.Empty(t, h.listingRepo.deleted)
	require.Len(t, h.listingRepo.updated, 2)
}

// TestSyncProductToChannels_variationsMode_updatesParentImagesNeverVariationImages
// proves the image gallery lands on the parent product's UpdateProduct
// call, never on either variation's UpdateVariation call -- Woo variations
// have no images of their own, they inherit the parent's.
func TestSyncProductToChannels_variationsMode_updatesParentImagesNeverVariationImages(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "902", VariantID: strPtr(variants[1].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	h.mediaRepo.byProduct = map[string][]pmsDomain.Media{
		product.ID: {{ID: "media-1", ProductID: &product.ID, URL: "https://cdn.example.com/front.jpg", Position: 0}},
	}
	uc := h.newSyncUseCase(t, 6)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	require.Equal(t, []integrationsDomain.WooProductImage{{URL: "https://cdn.example.com/front.jpg", Alt: "Carne Seca"}},
		h.wooClient.updateProductCalled[0].Product.Images)
	require.Len(t, h.wooClient.updateVariationCalled, 2)
}

// TestSyncProductToChannels_updatesExistingSeparateProductsInPlaceWhenShapeUnchanged
// is separate-products mode's equivalent steady state.
func TestSyncProductToChannels_updatesExistingSeparateProductsInPlaceWhenShapeUnchanged(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "701", VariantID: strPtr(variants[0].ID)},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "702", VariantID: strPtr(variants[1].ID)},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	uc := h.newSyncUseCase(t, 6)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 2)
	require.Empty(t, h.wooClient.createCalled)
	require.Empty(t, h.wooClient.deleteProductCalled)
	require.Empty(t, h.listingRepo.deleted)
	require.Len(t, h.listingRepo.updated, 2)
}

// TestSyncProductToChannels_createsVariationForNewlyAddedVariant covers a
// third Variant added to an already-linked variations-mode product: only
// the new one gets a CreateVariation call, the existing two are untouched
// creates (only updated).
func TestSyncProductToChannels_createsRowForNewlyAddedVariant(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		// variant-2 has no listing yet -- added after the initial push.
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	h.wooClient.createVariationWithID = "903"
	uc := h.newSyncUseCase(t, 6)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.createVariationCalled, 1)
	require.Equal(t, "900", h.wooClient.createVariationCalled[0].ParentExternalID)
	require.Equal(t, "412g", h.wooClient.createVariationCalled[0].Variation.AttributeValue)
	require.Len(t, h.wooClient.updateVariationCalled, 1, "the pre-existing variant-1 row is updated, not recreated")
	require.Len(t, h.listingRepo.created, 1)
	require.Equal(t, "903", h.listingRepo.created[0].ExternalID)
}

// TestSyncProductToChannels_deletesVariationForRemovedVariant covers the
// opposite edge: a Variant deleted from the product removes its Woo
// variation and listing row too, without disturbing its siblings.
func TestSyncProductToChannels_deletesVariationForRemovedVariant(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	remainingVariant := variants[:1] // variant-2 was deleted from the product
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "902", VariantID: strPtr(variants[1].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, remainingVariant, skus, listings)
	uc := h.newSyncUseCase(t, 5)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.deleteVariationCalled, 1)
	require.Equal(t, "902", h.wooClient.deleteVariationCalled[0].VariationExternalID)
	require.Equal(t, "900", h.wooClient.deleteVariationCalled[0].ParentExternalID)
	require.Equal(t, []string{"listing-2"}, h.listingRepo.deleted)
	require.Len(t, h.wooClient.updateVariationCalled, 1, "the surviving variant-1 row is still synced")
}

func TestSyncProductToChannels_skipsListingsForUnimplementedProviders(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: "shopify", ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 1)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled)
	require.Empty(t, h.listingRepo.updated)
}

type stubBolProductPusher struct {
	calls []appchannels.PushProductToBolInput
	err   error
}

func (s *stubBolProductPusher) Execute(_ context.Context, in appchannels.PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	return []channelsDomain.ProductChannelListing{{ID: "listing-bol", ProductID: in.ProductID, ConnectionID: in.ConnectionID}}, nil
}

// TestSyncProductToChannels_dispatchesBolListingsToBolPusher proves the
// additive dispatch branch added alongside the existing Woo shape logic --
// a "bol" listing routes to bolPusher.Execute instead of falling through
// to (or being silently skipped by) the Woo-only path below.
func TestSyncProductToChannels_dispatchesBolListingsToBolPusher(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "offer-1",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	bolPusher := &stubBolProductPusher{}
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 1), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{}, bolPusher, nil, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, bolPusher.calls, 1)
	require.Equal(t, product.ID, bolPusher.calls[0].ProductID)
	require.Equal(t, "conn-1", bolPusher.calls[0].ConnectionID)
	require.Empty(t, h.wooClient.updateProductCalled, "a bol listing must never reach the woo client")
}

func TestSyncProductToChannels_writesASyncLogEntryOnSuccess(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, "listing-1", *entry.ListingID)
	require.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, product.Name, entry.Fields["name"])
	require.Equal(t, "19.99", entry.Fields["regular_price"])
	// First-ever entry for this listing -- everything shows as new (from "").
	require.Equal(t, channelsDomain.FieldChange{From: "", To: product.Name}, entry.Changes["name"])
}

// TestSyncProductToChannels_auditLogTracksSalePriceExpiry proves the sale
// price revert is visible in the sync-log audit trail, not just pushed to
// Woo silently -- wooProductToFields must include sale_price/sale_end_date
// or a revert like this would leave Woo correctly updated but the audit
// trail (what the SyncLog page shows) blind to what changed.
func TestSyncProductToChannels_auditLogTracksSalePriceExpiry(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	// Previous entry recorded the sale as active.
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{
			ListingID: strPtr("listing-1"),
			Fields: map[string]string{
				"name": product.Name, "description": product.Description, "status": "publish",
				"regular_price": "19.99", "sale_price": "14.99", "sale_end_date": "2026-08-01",
				"virtual": "false", "weight": "", "length": "", "width": "", "height": "",
			},
		}},
	}
	uc := h.newSyncUseCase(t, 4) // sku here has no sale price -- simulates post-expiry

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.FieldChange{From: "14.99", To: ""}, entry.Changes["sale_price"])
	require.Equal(t, channelsDomain.FieldChange{From: "2026-08-01", To: ""}, entry.Changes["sale_end_date"])
}

func TestSyncProductToChannels_diffsAgainstThePreviousSyncLogEntry(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{ListingID: strPtr("listing-1"), Fields: map[string]string{"name": "Old Name", "regular_price": "19.99"}}},
	}
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	changes := h.syncLogRepo.created[0].Changes
	require.Equal(t, channelsDomain.FieldChange{From: "Old Name", To: product.Name}, changes["name"])
	_, priceChanged := changes["regular_price"]
	require.False(t, priceChanged, "unchanged field must not appear in the diff")
}

func TestSyncProductToChannels_skipsWritingALogEntryWhenNothingChangedSinceLastSync(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	// Previous entry already has the exact fields this sync will produce
	// (same name/description/status/price/virtual) — e.g. a second
	// pms.product.changed event fired moments later by a sibling API call
	// (product update + SKU update from the same "Save" click) that didn't
	// touch anything this use case pushes.
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{
			ListingID: strPtr("listing-1"),
			Fields: map[string]string{
				"name": product.Name, "description": product.Description, "status": "publish",
				"regular_price": "19.99", "virtual": "false", "weight": "", "length": "", "width": "", "height": "",
			},
		}},
	}
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.syncLogRepo.created, "identical fields since last entry must not produce a new row")
}

// TestSyncProductToChannels_neverCreatesListingForBulkPoolVariant proves a
// TracksWeight variant is never pushed as a new listing during steady-state
// sync -- it's an internal weight ledger, not something to sell, even
// though it has its own Variant row like any sellable one.
func TestSyncProductToChannels_neverCreatesListingForBulkPoolVariant(t *testing.T) {
	product, variants, skus := bulkPlusTwoVariantProduct(false)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "701", VariantID: strPtr(variants[0].ID)},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "702", VariantID: strPtr(variants[1].ID)},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	uc := h.newSyncUseCase(t, 6)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 2, "only the two sellable variants are synced")
	require.Empty(t, h.wooClient.createCalled, "the bulk pool variant is never created as a new listing")
	require.Empty(t, h.listingRepo.created)
}

// TestSyncProductToChannels_cleansUpStrayListingForVariantThatBecameBulkPool
// covers a variant that was sellable and already synced, then had its SKU
// flipped to TracksWeight afterwards -- the sync must tear down its
// external listing (same "removed variant" cleanup path as an actually
// deleted variant), leaving its sellable siblings untouched.
func TestSyncProductToChannels_cleansUpStrayListingForVariantThatBecameBulkPool(t *testing.T) {
	product, variants, skus := bulkPlusTwoVariantProduct(false)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "701", VariantID: strPtr(variants[0].ID)},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "702", VariantID: strPtr(variants[1].ID)},
		{ID: "listing-bulk", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "703", VariantID: strPtr("variant-bulk")},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	uc := h.newSyncUseCase(t, 7)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 2, "the sellable siblings are still synced normally")
	require.Equal(t, []string{"703"}, h.wooClient.deleteProductCalled)
	require.Equal(t, []string{"listing-bulk"}, h.listingRepo.deleted)
}

func TestSyncProductToChannels_writesASyncLogEntryOnFailureToo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.updateProductErr = errors.New("woo unreachable")
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncLogStatusError, entry.Status)
	require.Contains(t, entry.ErrorMessage, "woo unreachable")
}

// ── gating: product leaves/returns to Active ────────────────────────────────

// TestSyncProductToChannels_gatesWooListingOnFirstNonActiveSync proves the
// active->draft/archived transition: the listing's current fields (plus the
// new draft status) are pushed to Woo exactly once, then the row is marked
// gated -- "change the status in the destination channel, e.g. change woo
// product to draft."
func TestSyncProductToChannels_gatesWooListingOnFirstNonActiveSync(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.Status = pmsDomain.ProductStatusDraft
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 10), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{}, nil, nil, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1, "current fields pushed one last time")
	call := h.wooClient.updateProductCalled[0]
	require.Equal(t, "555", call.ExternalID)
	require.Equal(t, "draft", call.Product.Status)
	require.Equal(t, product.Name, call.Product.Name, "full fields, not a status-only payload -- per the merchant's choice")

	got := h.listingRepo.listByProduct[product.ID][0]
	require.NotNil(t, got.GatedAt, "listing must be marked gated")
}

// TestSyncProductToChannels_skipsAlreadyGatedWooListing proves the second
// (and every further) sync while still non-active makes zero calls at all --
// "any future updates is ignored until the product goes back to active."
func TestSyncProductToChannels_skipsAlreadyGatedWooListing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.Status = pmsDomain.ProductStatusArchived
	gatedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced, GatedAt: &gatedAt,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 1)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled, "already gated -- zero channel calls")
	require.Empty(t, h.listingRepo.updated, "no write at all, not even a no-op one")
	require.Empty(t, h.syncLogRepo.created)
}

// TestSyncProductToChannels_ungatesWooListingOnReactivation proves the
// draft/archived->active transition: the gate clears, a full field sync
// (including the new Active status) runs immediately, and stock is force
// -caught-up for every SKU -- "any future updates is ignored until the
// product in commerceos goes back to active, including inventory sync."
func TestSyncProductToChannels_ungatesWooListingOnReactivation(t *testing.T) {
	product, variant, sku := simplePhysicalProduct() // Active
	gatedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced, GatedAt: &gatedAt,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	stockCatchUp := &stubStockForceSyncer{}
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 10), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{}, nil, nil, stockCatchUp, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	require.Equal(t, "publish", h.wooClient.updateProductCalled[0].Product.Status)

	got := h.listingRepo.listByProduct[product.ID][0]
	require.Nil(t, got.GatedAt, "gate cleared")

	require.Equal(t, []string{sku.ID}, stockCatchUp.calls, "stock force-synced for every SKU on reactivation")
}

// TestSyncProductToChannels_gatesBolListingViaOnHold proves the Bol
// equivalent of gating: an onHoldByRetailer=true toggle (not a full offer
// push -- Bol has no "draft" concept, only on-hold) rather than dispatching
// through the full bolPusher.
func TestSyncProductToChannels_gatesBolListingViaOnHold(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.Status = pmsDomain.ProductStatusDraft
	_ = sku
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "offer-1",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	bolPusher := &stubBolProductPusher{}
	bolGater := &stubBolOfferGater{}
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 10), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{}, bolPusher, bolGater, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, bolPusher.calls, "a gated bol listing must never go through the full offer push")
	require.Equal(t, []stubOnHoldCall{{ConnectionID: "conn-1", ExternalID: "offer-1", OnHold: true}}, bolGater.calls)

	got := h.listingRepo.listByProduct[product.ID][0]
	require.NotNil(t, got.GatedAt)
}

// TestSyncProductToChannels_skipsAlreadyGatedBolListing mirrors the Woo
// already-gated test: zero calls of any kind on a second sync while still
// non-active.
func TestSyncProductToChannels_skipsAlreadyGatedBolListing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.Status = pmsDomain.ProductStatusDraft
	gatedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "offer-1", GatedAt: &gatedAt,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	bolPusher := &stubBolProductPusher{}
	bolGater := &stubBolOfferGater{}
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 1), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{}, bolPusher, bolGater, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, bolPusher.calls)
	require.Empty(t, bolGater.calls)
}

// TestSyncProductToChannels_ungatesBolListingOnReactivation proves the Bol
// reactivation path: onHoldByRetailer flips back off, the gate clears, the
// full offer push runs (via bolPusher, same as a normal sync), and stock is
// force-caught-up.
func TestSyncProductToChannels_ungatesBolListingOnReactivation(t *testing.T) {
	product, variant, sku := simplePhysicalProduct() // Active
	gatedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "offer-1", GatedAt: &gatedAt,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	bolPusher := &stubBolProductPusher{}
	bolGater := &stubBolOfferGater{}
	stockCatchUp := &stubStockForceSyncer{}
	uc := appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, 10), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo,
		&stubCategoryLinkResolver{}, bolPusher, bolGater, stockCatchUp, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Equal(t, []stubOnHoldCall{{ConnectionID: "conn-1", ExternalID: "offer-1", OnHold: false}}, bolGater.calls)
	require.Len(t, bolPusher.calls, 1, "full offer push runs immediately on reactivation")
	require.Equal(t, []string{sku.ID}, stockCatchUp.calls)

	got := h.listingRepo.listByProduct[product.ID][0]
	require.Nil(t, got.GatedAt)
}
