package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func (f *packFixture) repriceUseCase(t *testing.T, updater *fakeSKUUpdater) *apppms.RepriceWeightPackagesUseCase {
	return apppms.NewRepriceWeightPackagesUseCase(newMockGormDBN(t, 2), fakePackSKUs{f.catalog}, fakePackVariants{f.catalog}, updater)
}

func TestRepriceWeightPackages_setsEveryPackageToTheNewRatesPackagePrice(t *testing.T) {
	f := newPackFixture()
	f.setBulk(func(s *pmsDomain.SKU) { rate := int64(3299); s.PricePerKiloMinorUnits = &rate })
	f.addPackage("sku-380", 380)
	f.addPackage("sku-540", 540)
	f.catalog.skus[1].PriceMinorUnits = 1139 // old 29.99/kg price
	updater := &fakeSKUUpdater{}

	result, err := f.repriceUseCase(t, updater).Execute(context.Background(), "sku-bulk-1")

	require.NoError(t, err)
	assert.Equal(t, 2, result.Repriced)
	require.Len(t, updater.calls, 2)
	// 32.99/kg x 380g = 12.54 -> 12.49
	assert.Equal(t, "sku-380", updater.calls[0].ID)
	assert.Equal(t, int64(1249), updater.calls[0].PriceMinorUnits)
	assert.Equal(t, pmsDomain.SKUPricingModeFixed, updater.calls[0].PricingMode)
	require.NotNil(t, updater.calls[0].PortionedFromSKUID, "everything else passes through")
	// 32.99/kg x 540g = 17.81 -> 17.79
	assert.Equal(t, int64(1779), updater.calls[1].PriceMinorUnits)
}

func TestRepriceWeightPackages_leavesPackagesAlreadyAtThePriceAlone(t *testing.T) {
	f := newPackFixture()
	f.setBulk(func(s *pmsDomain.SKU) { rate := int64(2999); s.PricePerKiloMinorUnits = &rate })
	f.addPackage("sku-380", 380)
	f.catalog.skus[1].PriceMinorUnits = 1139
	updater := &fakeSKUUpdater{}

	result, err := f.repriceUseCase(t, updater).Execute(context.Background(), "sku-bulk-1")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Repriced)
	assert.Empty(t, updater.calls, "a redelivered event changes nothing")
}

func TestRepriceWeightPackages_ignoresSKUsThatAreNotItsPackages(t *testing.T) {
	f := newPackFixture()
	f.addHandMadePackage("sku-unlinked", "~ 380g", "2994721003809", pmsDomain.SKUPricingModeFixed, 1150)
	updater := &fakeSKUUpdater{}

	_, err := f.repriceUseCase(t, updater).Execute(context.Background(), "sku-bulk-1")

	require.NoError(t, err)
	assert.Empty(t, updater.calls)
}

func TestRepriceWeightPackages_aPoolNoLongerScaleSoldIsANoOp(t *testing.T) {
	f := newPackFixture()
	f.setBulk(func(s *pmsDomain.SKU) { s.PricePerKiloMinorUnits = nil; s.PricingMode = pmsDomain.SKUPricingModeFixed })
	f.addPackage("sku-380", 380)
	updater := &fakeSKUUpdater{}

	result, err := f.repriceUseCase(t, updater).Execute(context.Background(), "sku-bulk-1")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Repriced)
	assert.Empty(t, updater.calls)
}

// A failed update must surface so the consumer retries -- packages already
// repriced are skipped next time.
func TestRepriceWeightPackages_reportsAFailedUpdateAfterTryingTheRest(t *testing.T) {
	f := newPackFixture()
	f.setBulk(func(s *pmsDomain.SKU) { rate := int64(3299); s.PricePerKiloMinorUnits = &rate })
	f.addPackage("sku-380", 380)
	f.addPackage("sku-540", 540)
	updater := &fakeSKUUpdater{failID: "sku-380"}

	result, err := f.repriceUseCase(t, updater).Execute(context.Background(), "sku-bulk-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARNE-SECA-380G")
	assert.Equal(t, 1, result.Repriced)
}
