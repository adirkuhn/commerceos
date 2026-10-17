package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type previewHarness struct {
	conn     integrationsDomain.Connection
	sku      pmsDomain.SKU
	override *channelsDomain.BolProductOverride
	vat      *fakeVATRates
	client   *fakeBolClient
}

func newPreviewHarness() *previewHarness {
	conn := bolConn()
	conn.BolShippingCostMinorUnits = 495
	cost := int64(1200)
	client := &fakeBolClient{}
	client.getCommissionFn = func(_ context.Context, ean string, _ float64) (integrationsDomain.BolCommission, error) {
		return integrationsDomain.BolCommission{EAN: ean, FixedAmount: 0.30, Percentage: 15}, nil
	}
	return &previewHarness{
		conn:   conn,
		sku:    pmsDomain.SKU{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 2500, CostPriceMinorUnits: &cost},
		vat:    &fakeVATRates{basisPoints: 900},
		client: client,
	}
}

func (h *previewHarness) useCase(t *testing.T, txCount int) *appchannels.PreviewBolPricingUseCase {
	t.Helper()
	return appchannels.NewPreviewBolPricingUseCase(
		newMockGormDB(t, txCount), &stubConnectionRepo{conn: &h.conn}, &fakeBolClientFactory{client: h.client}, testEncryptionSecret,
		&fakeSKUByIDRepo{skus: map[string]pmsDomain.SKU{"sku-1": h.sku}}, &fakeBolOverrideRepo{override: h.override}, h.vat,
	)
}

func TestPreviewBolPricing_savedSettings(t *testing.T) {
	h := newPreviewHarness()
	h.override = &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1", BundlePricingEnabled: true,
		BundleTiers: []channelsDomain.BolBundleTier{{Quantity: 2}},
	}

	res, err := h.useCase(t, 1).Execute(context.Background(), appchannels.PreviewBolPricingInput{ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1"})

	require.NoError(t, err)
	assert.Equal(t, "0630509892495", res.EAN)
	assert.Equal(t, int64(2500), res.RegularPriceMinorUnits)
	assert.Equal(t, int64(900), res.ProductVATBasisPoints)
	assert.Equal(t, int64(495), res.ShippingMinorUnits)
	require.Len(t, res.Levels, 2)
	assert.Equal(t, int64(3849), res.Levels[0].UnitPriceMinorUnits)
	assert.Equal(t, int64(3549), res.Levels[1].UnitPriceMinorUnits)
	require.NotNil(t, res.Levels[0].ProfitMinorUnits)
	assert.Equal(t, res.Levels[0].Economics.NetMinorUnits-1200, *res.Levels[0].ProfitMinorUnits, "what is left minus the cost price")
}

func TestPreviewBolPricing_draftReplacesTheSavedSettings(t *testing.T) {
	h := newPreviewHarness()
	custom := int64(3000)

	res, err := h.useCase(t, 1).Execute(context.Background(), appchannels.PreviewBolPricingInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1",
		Draft: &appchannels.BolPricingDraft{CustomPriceMinorUnits: &custom, BundlePricingEnabled: true},
	})

	require.NoError(t, err)
	require.Len(t, res.Levels, 5, "switching bulk prices on without tiers previews the 2/4/6/8 defaults")
	assert.Equal(t, int64(3000), res.Levels[0].UnitPriceMinorUnits)
	assert.False(t, res.Levels[0].Auto)
	assert.True(t, res.Levels[1].Auto)
}

func TestPreviewBolPricing_noCostPriceMeansNoProfit(t *testing.T) {
	h := newPreviewHarness()
	h.sku.CostPriceMinorUnits = nil

	res, err := h.useCase(t, 1).Execute(context.Background(), appchannels.PreviewBolPricingInput{ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1"})

	require.NoError(t, err)
	assert.Nil(t, res.Levels[0].ProfitMinorUnits)
}

func TestPreviewBolPricing_invalidDraftTiers(t *testing.T) {
	h := newPreviewHarness()

	_, err := h.useCase(t, 0).Execute(context.Background(), appchannels.PreviewBolPricingInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1",
		Draft: &appchannels.BolPricingDraft{BundlePricingEnabled: true, BundleTiers: []channelsDomain.BolBundleTier{{Quantity: 1}}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrBolInvalidBundleTiers)
}

func TestPreviewBolPricing_pricesBreakingTheFivePercentStep(t *testing.T) {
	h := newPreviewHarness()
	custom, tier := int64(3000), int64(2900)

	_, err := h.useCase(t, 1).Execute(context.Background(), appchannels.PreviewBolPricingInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1",
		Draft: &appchannels.BolPricingDraft{CustomPriceMinorUnits: &custom, BundlePricingEnabled: true,
			BundleTiers: []channelsDomain.BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: &tier}}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrBolBundlePriceStep)
}

func TestPreviewBolPricing_invalidEAN(t *testing.T) {
	h := newPreviewHarness()
	h.sku.Barcode = "12345"

	_, err := h.useCase(t, 1).Execute(context.Background(), appchannels.PreviewBolPricingInput{ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1"})

	assert.ErrorIs(t, err, channelsDomain.ErrBolInvalidEAN)
}
