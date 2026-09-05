package whatsapp_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeSKURepo struct {
	pmsDomain.SKURepository
	byCode map[string]pmsDomain.SKU
}

func (f *fakeSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if s, ok := f.byCode[code]; ok {
		return &s, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}

type fakeVariantRepo struct {
	pmsDomain.VariantRepository
	byID map[string]pmsDomain.Variant
}

func (f *fakeVariantRepo) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	if v, ok := f.byID[id]; ok {
		return &v, nil
	}
	return nil, pmsDomain.ErrVariantNotFound
}

type fakeProductRepo struct {
	pmsDomain.ProductRepository
	byID map[string]pmsDomain.Product
}

func (f *fakeProductRepo) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if p, ok := f.byID[id]; ok {
		return &p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}

type fakeSalesChannelRepo struct {
	channelsDomain.SalesChannelRepository
	channels []channelsDomain.SalesChannel
}

func (f *fakeSalesChannelRepo) ListByTenant(_ context.Context) ([]channelsDomain.SalesChannel, error) {
	return f.channels, nil
}

type fakeOverrideRepo struct {
	channelsDomain.SalesChannelProductOverrideRepository
	override *channelsDomain.SalesChannelProductOverride
	err      error
}

func (f *fakeOverrideRepo) FindByChannelAndProduct(_ context.Context, _, _ string) (*channelsDomain.SalesChannelProductOverride, error) {
	return f.override, f.err
}

type fakeStockLevelGetter struct {
	level *appinventory.StockLevel
	err   error
}

func (f *fakeStockLevelGetter) Execute(_ context.Context, _ string) (*appinventory.StockLevel, error) {
	return f.level, f.err
}

// ── test setup ───────────────────────────────────────────────────────────────

const testWhatsAppChannelID = "channel-1"

func newLookupUseCase(t *testing.T, sku pmsDomain.SKU, variant pmsDomain.Variant, product pmsDomain.Product, level *appinventory.StockLevel) (*appwhatsapp.LookupProductInfoUseCase, *fakeOverrideRepo) {
	t.Helper()
	overrideRepo := &fakeOverrideRepo{}
	uc := appwhatsapp.NewLookupProductInfoUseCase(
		newMockGormDB(t, 1),
		&fakeSKURepo{byCode: map[string]pmsDomain.SKU{sku.SKUCode: sku}},
		&fakeVariantRepo{byID: map[string]pmsDomain.Variant{variant.ID: variant}},
		&fakeProductRepo{byID: map[string]pmsDomain.Product{product.ID: product}},
		&fakeSalesChannelRepo{channels: []channelsDomain.SalesChannel{
			{ID: testWhatsAppChannelID, Type: channelsDomain.ChannelTypeWhatsApp, Active: true, EnrollmentMode: channelsDomain.EnrollmentModeAll},
		}},
		overrideRepo,
		&fakeStockLevelGetter{level: level},
	)
	return uc, overrideRepo
}

func TestLookupProductInfoUseCase_enrolledProduct_returnsFullInfo(t *testing.T) {
	sku := pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SKUCode: "ABC123", PriceMinorUnits: 1999, CurrencyCode: "EUR"}
	variant := pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}
	product := pmsDomain.Product{ID: "product-1", Name: "Widget"}
	expiry := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	uc, _ := newLookupUseCase(t, sku, variant, product, &appinventory.StockLevel{
		Syncable: 42, SourceBatch: &appinventory.BatchSyncInfo{LotNumber: "LOT-1", ExpiryDate: expiry},
	})

	info, err := uc.Execute(context.Background(), "ABC123")

	require.NoError(t, err)
	assert.Equal(t, "Widget", info.ProductName)
	assert.Equal(t, "ABC123", info.SKUCode)
	assert.Equal(t, int64(42), info.StockQuantity)
	assert.Equal(t, int64(1999), info.PriceMinorUnits)
	require.NotNil(t, info.NearestExpiry)
	assert.True(t, expiry.Equal(*info.NearestExpiry))
}

func TestLookupProductInfoUseCase_unknownSKUCode_returnsErrProductNotFound(t *testing.T) {
	uc, _ := newLookupUseCase(t, pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SKUCode: "ABC123"},
		pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}, pmsDomain.Product{ID: "product-1"}, &appinventory.StockLevel{})

	_, err := uc.Execute(context.Background(), "DOES-NOT-EXIST")

	require.ErrorIs(t, err, appwhatsapp.ErrProductNotFound)
}

func TestLookupProductInfoUseCase_productExcludedByOverride_returnsErrProductNotFound(t *testing.T) {
	sku := pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SKUCode: "ABC123"}
	variant := pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}
	product := pmsDomain.Product{ID: "product-1", Name: "Widget"}
	uc, overrideRepo := newLookupUseCase(t, sku, variant, product, &appinventory.StockLevel{})
	overrideRepo.override = &channelsDomain.SalesChannelProductOverride{Included: false}

	_, err := uc.Execute(context.Background(), "ABC123")

	require.ErrorIs(t, err, appwhatsapp.ErrProductNotFound, "an explicit exclude override must win over EnrollmentModeAll, same as every other channel")
}

func TestLookupProductInfoUseCase_manualModeChannel_excludesUnlistedProduct(t *testing.T) {
	sku := pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SKUCode: "ABC123"}
	variant := pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}
	product := pmsDomain.Product{ID: "product-1", Name: "Widget"}
	overrideRepo := &fakeOverrideRepo{}
	uc := appwhatsapp.NewLookupProductInfoUseCase(
		newMockGormDB(t, 1),
		&fakeSKURepo{byCode: map[string]pmsDomain.SKU{sku.SKUCode: sku}},
		&fakeVariantRepo{byID: map[string]pmsDomain.Variant{variant.ID: variant}},
		&fakeProductRepo{byID: map[string]pmsDomain.Product{product.ID: product}},
		&fakeSalesChannelRepo{channels: []channelsDomain.SalesChannel{
			{ID: testWhatsAppChannelID, Type: channelsDomain.ChannelTypeWhatsApp, Active: true, EnrollmentMode: channelsDomain.EnrollmentModeManual},
		}},
		overrideRepo,
		&fakeStockLevelGetter{level: &appinventory.StockLevel{}},
	)

	_, err := uc.Execute(context.Background(), "ABC123")

	require.ErrorIs(t, err, appwhatsapp.ErrProductNotFound)
}

func TestLookupProductInfoUseCase_noWhatsAppChannel_returnsErrNoWhatsAppChannel(t *testing.T) {
	uc := appwhatsapp.NewLookupProductInfoUseCase(
		newMockGormDB(t, 1),
		&fakeSKURepo{}, &fakeVariantRepo{}, &fakeProductRepo{},
		&fakeSalesChannelRepo{channels: nil},
		&fakeOverrideRepo{},
		&fakeStockLevelGetter{},
	)

	_, err := uc.Execute(context.Background(), "ABC123")

	require.ErrorIs(t, err, appwhatsapp.ErrNoWhatsAppChannel)
}

func TestLookupProductInfoUseCase_inactiveWhatsAppChannel_treatedAsNoChannel(t *testing.T) {
	uc := appwhatsapp.NewLookupProductInfoUseCase(
		newMockGormDB(t, 1),
		&fakeSKURepo{}, &fakeVariantRepo{}, &fakeProductRepo{},
		&fakeSalesChannelRepo{channels: []channelsDomain.SalesChannel{
			{ID: testWhatsAppChannelID, Type: channelsDomain.ChannelTypeWhatsApp, Active: false},
		}},
		&fakeOverrideRepo{},
		&fakeStockLevelGetter{},
	)

	_, err := uc.Execute(context.Background(), "ABC123")

	require.ErrorIs(t, err, appwhatsapp.ErrNoWhatsAppChannel)
}
