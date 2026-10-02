package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type fakeSKUUpdater struct {
	calls  []apppms.UpdateSKUInput
	failID string
}

func (f *fakeSKUUpdater) Execute(_ context.Context, in apppms.UpdateSKUInput) (*pmsDomain.SKU, error) {
	if in.ID == f.failID {
		return nil, errors.New("boom")
	}
	f.calls = append(f.calls, in)
	return &pmsDomain.SKU{ID: in.ID}, nil
}

// linkFixture is the real Carne Seca setup: the "por kilo" variant is the
// pool (scale code 2994721, EUR 29.99/kg), and variants "~ 380g" etc. hold
// packages made by hand, barcoded with the scale's own label.
func linkFixture() *packFixture {
	f := newPackFixture()
	f.catalog.skus[0].SKUCode = "WOO-17522"
	return f
}

func (f *packFixture) addHandMadePackage(id, variantName, barcode string, mode pmsDomain.SKUPricingMode, price int64) {
	variantID := "variant-" + id
	f.catalog.variants = append(f.catalog.variants, pmsDomain.Variant{ID: variantID, ProductID: "product-1", Name: variantName})
	sku := pmsDomain.SKU{
		ID: id, VariantID: variantID, SKUCode: barcode, Barcode: barcode, PriceMinorUnits: price, CurrencyCode: "EUR",
		UnitType: pmsDomain.SKUUnitTypeUnit, PricingMode: mode, Status: pmsDomain.SKUStatusActive,
	}
	if mode == pmsDomain.SKUPricingModePerWeight {
		perKilo := int64(29990)
		sku.PricePerKiloMinorUnits = &perKilo
	}
	f.catalog.skus = append(f.catalog.skus, sku)
}

func (f *packFixture) linkUseCase(t *testing.T, updater *fakeSKUUpdater) *apppms.LinkExistingWeightPackagesUseCase {
	return apppms.NewLinkExistingWeightPackagesUseCase(newMockGormDBN(t, 2), fakePackSKUs{f.catalog}, fakePackVariants{f.catalog}, updater)
}

func TestLinkExistingWeightPackages_linksEachScaleLabelledVariantWithItsWeight(t *testing.T) {
	f := linkFixture()
	f.addHandMadePackage("sku-380", "~ 380g", "2994721003809", pmsDomain.SKUPricingModeFixed, 1150)
	f.addHandMadePackage("sku-404", "~ 404g", "2994721004042", pmsDomain.SKUPricingModeFixed, 1212)
	updater := &fakeSKUUpdater{}

	result, err := f.linkUseCase(t, updater).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1"})

	require.NoError(t, err)
	require.Len(t, result.Linked, 2)
	assert.Equal(t, int64(380), result.Linked[0].WeightGrams)
	assert.Equal(t, "~ 380g", result.Linked[0].VariantName)
	require.Len(t, updater.calls, 2)
	in := updater.calls[0]
	assert.Equal(t, "sku-380", in.ID)
	require.NotNil(t, in.PortionedFromSKUID)
	assert.Equal(t, "sku-bulk-1", *in.PortionedFromSKUID)
	require.NotNil(t, in.WeightGrams)
	assert.Equal(t, int64(380), *in.WeightGrams)
	assert.Equal(t, pmsDomain.SKUUnitTypeUnit, in.UnitType)
	assert.Equal(t, int64(1150), in.PriceMinorUnits, "a fixed price you set by hand is kept")
	assert.False(t, result.Linked[0].PriceChanged)
	assert.Equal(t, "2994721003809", in.Barcode, "barcode and everything else pass through untouched")
	assert.Empty(t, result.Skipped)
}

// A package priced per weight would charge by scale at checkout -- but a
// package's label already carries its weight; Woo needs a fixed price.
func TestLinkExistingWeightPackages_givesAPerWeightPackageItsFixedPrice(t *testing.T) {
	f := linkFixture()
	f.addHandMadePackage("sku-528", "~ 528g", "2994721005285", pmsDomain.SKUPricingModePerWeight, 2999)
	updater := &fakeSKUUpdater{}

	result, err := f.linkUseCase(t, updater).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1"})

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	in := updater.calls[0]
	assert.Equal(t, pmsDomain.SKUPricingModeFixed, in.PricingMode)
	assert.Nil(t, in.PricePerKiloMinorUnits)
	// 29990/kg * 528g = 15834.72 -> 15835
	assert.Equal(t, int64(15835), in.PriceMinorUnits)
	assert.True(t, result.Linked[0].PriceChanged)
	assert.Equal(t, int64(15835), result.Linked[0].PriceMinorUnits)
}

func TestLinkExistingWeightPackages_readsTheWeightFromTheSKUCodeWhenTheBarcodeIsBlank(t *testing.T) {
	f := linkFixture()
	f.addHandMadePackage("sku-380", "~ 380g", "2994721003809", pmsDomain.SKUPricingModeFixed, 1150)
	f.catalog.skus[len(f.catalog.skus)-1].Barcode = ""
	updater := &fakeSKUUpdater{}

	result, err := f.linkUseCase(t, updater).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1"})

	require.NoError(t, err)
	require.Len(t, result.Linked, 1)
	assert.Equal(t, int64(380), result.Linked[0].WeightGrams)
}

func TestLinkExistingWeightPackages_skipsWhatIsNotOneOfThisPoolsPackages(t *testing.T) {
	f := linkFixture()
	f.addHandMadePackage("sku-other-scale", "~ 400g", "2994722004003", pmsDomain.SKUPricingModeFixed, 1000)
	f.addHandMadePackage("sku-ean", "Gift box", "7897122604154", pmsDomain.SKUPricingModeFixed, 1000)
	f.addPackage("sku-already", 442) // already linked to this pool: nothing to do
	otherPool := "sku-other-pool"
	f.addHandMadePackage("sku-other-pool-pkg", "~ 450g", "2994721004509", pmsDomain.SKUPricingModeFixed, 1000)
	f.catalog.skus[len(f.catalog.skus)-1].PortionedFromSKUID = &otherPool
	f.addHandMadePackage("sku-twin-a", "~ 460g", "2994721004608", pmsDomain.SKUPricingModeFixed, 1000)
	f.catalog.skus = append(f.catalog.skus, pmsDomain.SKU{ID: "sku-twin-b", VariantID: "variant-sku-twin-a", Barcode: "2994721004608"})
	updater := &fakeSKUUpdater{}

	result, err := f.linkUseCase(t, updater).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1"})

	require.NoError(t, err)
	assert.Empty(t, result.Linked)
	assert.Empty(t, updater.calls)
	reasons := map[string]apppms.LinkSkipReason{}
	for _, s := range result.Skipped {
		reasons[s.SKU.ID] = s.Reason
	}
	assert.Equal(t, map[string]apppms.LinkSkipReason{
		"sku-other-scale":    apppms.LinkSkipOtherScaleCode,
		"sku-ean":            apppms.LinkSkipNotAScaleLabel,
		"sku-other-pool-pkg": apppms.LinkSkipLinkedToAnotherPool,
		"sku-twin-a":         apppms.LinkSkipSeveralSKUs,
		"sku-twin-b":         apppms.LinkSkipSeveralSKUs,
	}, reasons)
}

func TestLinkExistingWeightPackages_dryRunPreviewsWithoutChangingAnything(t *testing.T) {
	f := linkFixture()
	f.addHandMadePackage("sku-380", "~ 380g", "2994721003809", pmsDomain.SKUPricingModeFixed, 1150)
	updater := &fakeSKUUpdater{}

	result, err := f.linkUseCase(t, updater).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1", DryRun: true})

	require.NoError(t, err)
	require.Len(t, result.Linked, 1)
	assert.Empty(t, updater.calls)
}

func TestLinkExistingWeightPackages_oneFailingUpdateDoesNotStopTheRest(t *testing.T) {
	f := linkFixture()
	f.addHandMadePackage("sku-380", "~ 380g", "2994721003809", pmsDomain.SKUPricingModeFixed, 1150)
	f.addHandMadePackage("sku-404", "~ 404g", "2994721004042", pmsDomain.SKUPricingModeFixed, 1212)
	updater := &fakeSKUUpdater{failID: "sku-380"}

	result, err := f.linkUseCase(t, updater).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1"})

	require.NoError(t, err)
	require.Len(t, result.Linked, 1)
	assert.Equal(t, "sku-404", result.Linked[0].SKU.ID)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "2994721003809")
}

func TestLinkExistingWeightPackages_rejectsAPoolThatIsNotScaleSold(t *testing.T) {
	f := linkFixture()
	f.setBulk(func(s *pmsDomain.SKU) { s.ScaleCode = nil })

	_, err := f.linkUseCase(t, &fakeSKUUpdater{}).Execute(context.Background(), apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1"})

	assert.ErrorIs(t, err, pmsDomain.ErrBulkSKUNotWeightPool)
}
