package pms_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type fakeBatchReader struct {
	byID  map[string]invDomain.Batch
	bySKU map[string][]invDomain.Batch
}

func (f *fakeBatchReader) FindByID(_ context.Context, id string) (*invDomain.Batch, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, invDomain.ErrBatchNotFound
	}
	return &b, nil
}

func (f *fakeBatchReader) ListBySKU(_ context.Context, skuID string) ([]invDomain.Batch, error) {
	return f.bySKU[skuID], nil
}

// fakePackCatalog is an in-memory product -> variants -> SKUs tree serving
// the use case's SKU/variant/product reader ports.
type fakePackCatalog struct {
	products map[string]pmsDomain.Product
	variants []pmsDomain.Variant
	skus     []pmsDomain.SKU
}

func (c *fakePackCatalog) FindSKU(id string) (*pmsDomain.SKU, error) {
	for _, s := range c.skus {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, pmsDomain.ErrSKUNotFound
}

type fakePackSKUs struct{ *fakePackCatalog }

func (f fakePackSKUs) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	return f.FindSKU(id)
}
func (f fakePackSKUs) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	var out []pmsDomain.SKU
	for _, s := range f.skus {
		if s.VariantID == variantID {
			out = append(out, s)
		}
	}
	return out, nil
}

type fakePackVariants struct{ *fakePackCatalog }

func (f fakePackVariants) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	for _, v := range f.variants {
		if v.ID == id {
			return &v, nil
		}
	}
	return nil, pmsDomain.ErrVariantNotFound
}
func (f fakePackVariants) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	var out []pmsDomain.Variant
	for _, v := range f.variants {
		if v.ProductID == productID {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakePackProducts struct{ *fakePackCatalog }

func (f fakePackProducts) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, pmsDomain.ErrProductNotFound
	}
	return &p, nil
}

type fakeVariantCreator struct {
	created []apppms.CreateVariantInput
}

func (f *fakeVariantCreator) Execute(_ context.Context, in apppms.CreateVariantInput) (*pmsDomain.Variant, error) {
	f.created = append(f.created, in)
	return &pmsDomain.Variant{ID: fmt.Sprintf("variant-new-%d", len(f.created)), ProductID: in.ProductID, Name: in.Name, Position: in.Position}, nil
}

type fakeProductUpdater struct {
	calls []apppms.UpdateProductInput
}

func (f *fakeProductUpdater) Execute(_ context.Context, in apppms.UpdateProductInput) (*pmsDomain.Product, error) {
	f.calls = append(f.calls, in)
	return &pmsDomain.Product{ID: in.ID, HasVariants: in.HasVariants}, nil
}

type fakeSKUCreator struct {
	created []apppms.CreateSKUInput
	err     error
}

func (f *fakeSKUCreator) Execute(_ context.Context, in apppms.CreateSKUInput) (*pmsDomain.SKU, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.created = append(f.created, in)
	sku := pmsDomain.SKU{
		ID: fmt.Sprintf("sku-portion-%d", len(f.created)), VariantID: in.VariantID, SKUCode: in.SKUCode, Barcode: in.Barcode,
		PriceMinorUnits: in.PriceMinorUnits, CurrencyCode: in.CurrencyCode, WeightGrams: in.WeightGrams,
		UnitType: in.UnitType, PortionedFromSKUID: in.PortionedFromSKUID, PricingMode: in.PricingMode, Status: in.Status,
	}
	return &sku, nil
}

type fakeBatchCreator struct {
	created []invapp.CreateBatchInput
}

func (f *fakeBatchCreator) Execute(_ context.Context, in invapp.CreateBatchInput) (*invDomain.Batch, error) {
	f.created = append(f.created, in)
	return &invDomain.Batch{ID: fmt.Sprintf("batch-new-%d", len(f.created)), SKUID: in.SKUID, WarehouseID: in.WarehouseID, LotNumber: in.LotNumber, ExpiryDate: in.ExpiryDate}, nil
}

type fakePortioner struct {
	calls []invapp.PortionFromBulkInput
	err   error
}

func (f *fakePortioner) Execute(_ context.Context, in invapp.PortionFromBulkInput) (*invapp.PortionFromBulkResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.calls = append(f.calls, in)
	return &invapp.PortionFromBulkResult{}, nil
}

type fakeScopedStock struct {
	grams  int64
	gotSKU string
	gotWH  string
	gotLot *string
}

func (f *fakeScopedStock) Execute(_ context.Context, skuID, warehouseID string, batchID *string) (int64, error) {
	f.gotSKU, f.gotWH, f.gotLot = skuID, warehouseID, batchID
	return f.grams, nil
}

func weightPoolBulkSKU() pmsDomain.SKU {
	scaleCode := "2994721"
	pricePerKilo := int64(29990) // EUR 29.99/kg in minor units
	return pmsDomain.SKU{
		ID: "sku-bulk-1", VariantID: "variant-1", SKUCode: "CARNE-SECA", CurrencyCode: "EUR",
		UnitType: pmsDomain.SKUUnitTypeKilo, PricingMode: pmsDomain.SKUPricingModePerWeight,
		PricePerKiloMinorUnits: &pricePerKilo, ScaleCode: &scaleCode, Status: pmsDomain.SKUStatusActive,
	}
}

type packFixture struct {
	catalog        *fakePackCatalog
	batches        *fakeBatchReader
	variantCreator *fakeVariantCreator
	productUpdater *fakeProductUpdater
	skuCreator     *fakeSKUCreator
	batchCreator   *fakeBatchCreator
	portioner      *fakePortioner
	stock          *fakeScopedStock
}

// newPackFixture is the Carne Seca product already in variants mode, its
// only variant holding the bulk weight-pool SKU.
func newPackFixture() *packFixture {
	return &packFixture{
		catalog: &fakePackCatalog{
			products: map[string]pmsDomain.Product{"product-1": {
				ID: "product-1", Name: "Carne Seca", Status: pmsDomain.ProductStatusActive, HasVariants: true,
			}},
			variants: []pmsDomain.Variant{{ID: "variant-1", ProductID: "product-1", Name: "Bulk"}},
			skus:     []pmsDomain.SKU{weightPoolBulkSKU()},
		},
		batches:        &fakeBatchReader{byID: map[string]invDomain.Batch{}, bySKU: map[string][]invDomain.Batch{}},
		variantCreator: &fakeVariantCreator{},
		productUpdater: &fakeProductUpdater{},
		skuCreator:     &fakeSKUCreator{},
		batchCreator:   &fakeBatchCreator{},
		portioner:      &fakePortioner{},
		stock:          &fakeScopedStock{grams: 50000},
	}
}

// addPackage puts an existing package SKU off the bulk pool on its own
// variant, the way this use case creates them.
func (f *packFixture) addPackage(skuID string, weight int64) {
	variantID := "variant-of-" + skuID
	bulkID := "sku-bulk-1"
	w := weight
	f.catalog.variants = append(f.catalog.variants, pmsDomain.Variant{ID: variantID, ProductID: "product-1", Name: fmt.Sprintf("~%dg", weight)})
	f.catalog.skus = append(f.catalog.skus, pmsDomain.SKU{
		ID: skuID, VariantID: variantID, SKUCode: fmt.Sprintf("CARNE-SECA-%dG", weight), WeightGrams: &w,
		UnitType: pmsDomain.SKUUnitTypeUnit, PortionedFromSKUID: &bulkID, Status: pmsDomain.SKUStatusActive,
	})
}

func (f *packFixture) setBulk(mutate func(*pmsDomain.SKU)) {
	mutate(&f.catalog.skus[0])
}

func (f *packFixture) useCase(t *testing.T) *apppms.PackWeightPortionsUseCase {
	return apppms.NewPackWeightPortionsUseCase(newMockGormDBN(t, 10), apppms.PackWeightPortionsDeps{
		SKUs: fakePackSKUs{f.catalog}, Variants: fakePackVariants{f.catalog}, Products: fakePackProducts{f.catalog},
		Batches: f.batches, CreateVariant: f.variantCreator, EnableVariants: f.productUpdater,
		CreateSKU: f.skuCreator, CreateBatch: f.batchCreator, Portion: f.portioner, Stock: f.stock,
	})
}

func lines(pairs ...int64) []apppms.WeightPackageLine {
	out := make([]apppms.WeightPackageLine, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, apppms.WeightPackageLine{Quantity: pairs[i], WeightGrams: pairs[i+1]})
	}
	return out
}

func TestPackWeightPortions_createsOneSKUPerDistinctWeightStockedWithItsQuantity(t *testing.T) {
	f := newPackFixture()

	result, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(1, 430, 4, 540, 1, 412, 10, 380),
	})

	require.NoError(t, err)
	require.Len(t, f.skuCreator.created, 4, "one SKU per distinct weight, not one per package")
	require.Len(t, result.Lines, 4)
	assert.Equal(t, []int64{430, 540, 412, 380}, []int64{
		result.Lines[0].WeightGrams, result.Lines[1].WeightGrams, result.Lines[2].WeightGrams, result.Lines[3].WeightGrams,
	})
	require.Len(t, f.portioner.calls, 4)
	assert.Equal(t, int64(10), f.portioner.calls[3].Quantity)
	assert.Equal(t, int64(380), *f.portioner.calls[3].WeightGrams)
	assert.True(t, result.Lines[3].Created)
	assert.Equal(t, int64(10), result.Lines[3].Quantity)

	require.Len(t, f.variantCreator.created, 4, "each package weight is its own variant -- channels push one SKU per variant")
	assert.Equal(t, "~380g", f.variantCreator.created[3].Name)
	assert.Equal(t, "product-1", f.variantCreator.created[3].ProductID)
	assert.Empty(t, f.productUpdater.calls, "already in variants mode")

	sku380 := f.skuCreator.created[3]
	assert.Equal(t, "variant-new-4", sku380.VariantID)
	assert.Equal(t, "CARNE-SECA-380G", sku380.SKUCode)
	assert.Equal(t, pmsDomain.SKUUnitTypeUnit, sku380.UnitType)
	assert.Equal(t, pmsDomain.SKUPricingModeFixed, sku380.PricingMode)
	require.NotNil(t, sku380.PortionedFromSKUID)
	assert.Equal(t, "sku-bulk-1", *sku380.PortionedFromSKUID)
	// 29990/kg * 380g / 1000 = 11396.2 -> 11396, same rounding as checkout.
	assert.Equal(t, int64(11396), sku380.PriceMinorUnits)
	decoded, err := pmsDomain.DecodeWeightEmbeddedBarcode(sku380.Barcode)
	require.NoError(t, err)
	assert.Equal(t, "2994721", decoded.ScaleCode)
	assert.Equal(t, int64(380), decoded.WeightGrams)
}

func TestPackWeightPortions_mergesRepeatedWeightsIntoOneLine(t *testing.T) {
	f := newPackFixture()

	result, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(3, 380, 1, 412, 2, 380),
	})

	require.NoError(t, err)
	require.Len(t, f.skuCreator.created, 2, "the old tool failed here with 'sku code already exists'")
	require.Len(t, result.Lines, 2)
	assert.Equal(t, int64(380), result.Lines[0].WeightGrams)
	assert.Equal(t, int64(5), result.Lines[0].Quantity)
}

func TestPackWeightPortions_restocksTheExistingPackageSKUForAWeight(t *testing.T) {
	f := newPackFixture()
	f.addPackage("sku-existing-380", 380)
	// Same weight, but portioned off a different pool -- not a match.
	otherW, otherBulk := int64(380), "sku-other-bulk"
	f.catalog.variants = append(f.catalog.variants, pmsDomain.Variant{ID: "variant-other", ProductID: "product-1"})
	f.catalog.skus = append(f.catalog.skus, pmsDomain.SKU{ID: "sku-other-pool-380", VariantID: "variant-other", WeightGrams: &otherW, PortionedFromSKUID: &otherBulk})

	result, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(3, 380),
	})

	require.NoError(t, err)
	assert.Empty(t, f.skuCreator.created)
	assert.Empty(t, f.variantCreator.created)
	require.Len(t, f.portioner.calls, 1)
	assert.Equal(t, "sku-existing-380", f.portioner.calls[0].TargetSKUID)
	assert.False(t, result.Lines[0].Created)
	assert.Equal(t, "sku-existing-380", result.Lines[0].SKU.ID)
}

// TestPackWeightPortions_ignoresAPackageSharingThePoolsOwnVariant: the
// previous tool created package SKUs on the bulk SKU's own variant, which
// no sales channel can push (one SKU per variant) -- never restock those.
func TestPackWeightPortions_ignoresAPackageSharingThePoolsOwnVariant(t *testing.T) {
	f := newPackFixture()
	w, bulkID := int64(380), "sku-bulk-1"
	f.catalog.skus = append(f.catalog.skus, pmsDomain.SKU{
		ID: "sku-legacy-380", VariantID: "variant-1", SKUCode: "OLD-380G", WeightGrams: &w, PortionedFromSKUID: &bulkID, Status: pmsDomain.SKUStatusActive,
	})

	result, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(1, 380),
	})

	require.NoError(t, err)
	require.Len(t, f.skuCreator.created, 1)
	assert.True(t, result.Lines[0].Created)
}

func TestPackWeightPortions_carriesTheLotsExpiryOntoANewPackageBatch(t *testing.T) {
	f := newPackFixture()
	expiry := time.Date(2030, 10, 10, 0, 0, 0, 0, time.UTC)
	f.batches.byID["batch-lot-1"] = invDomain.Batch{ID: "batch-lot-1", SKUID: "sku-bulk-1", LotNumber: "LOT-1", ExpiryDate: &expiry}
	lot := "batch-lot-1"

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BulkBatchID: &lot, Lines: lines(2, 430),
	})

	require.NoError(t, err)
	require.Len(t, f.batchCreator.created, 1)
	assert.Equal(t, "LOT-1", f.batchCreator.created[0].LotNumber)
	assert.Equal(t, &expiry, f.batchCreator.created[0].ExpiryDate)
	assert.Equal(t, "warehouse-1", f.batchCreator.created[0].WarehouseID)
	require.NotNil(t, f.portioner.calls[0].TargetBatchID)
	assert.Equal(t, "batch-new-1", *f.portioner.calls[0].TargetBatchID)
	require.NotNil(t, f.portioner.calls[0].BatchID)
	assert.Equal(t, "batch-lot-1", *f.portioner.calls[0].BatchID, "draws down the BULK sku's own lot")
}

func TestPackWeightPortions_reusesTheExistingPackageBatchForTheSameLot(t *testing.T) {
	f := newPackFixture()
	expiry := time.Date(2030, 10, 10, 0, 0, 0, 0, time.UTC)
	sameDay := time.Date(2030, 10, 10, 0, 0, 0, 0, time.UTC)
	f.batches.byID["batch-lot-1"] = invDomain.Batch{ID: "batch-lot-1", SKUID: "sku-bulk-1", LotNumber: "LOT-1", ExpiryDate: &expiry}
	f.addPackage("sku-existing-380", 380)
	f.batches.bySKU["sku-existing-380"] = []invDomain.Batch{
		{ID: "batch-other-wh", SKUID: "sku-existing-380", WarehouseID: "warehouse-2", LotNumber: "LOT-1", ExpiryDate: &sameDay, Status: invDomain.BatchStatusActive},
		{ID: "batch-pkg-lot-1", SKUID: "sku-existing-380", WarehouseID: "warehouse-1", LotNumber: "LOT-1", ExpiryDate: &sameDay, Status: invDomain.BatchStatusActive},
	}
	lot := "batch-lot-1"

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BulkBatchID: &lot, Lines: lines(3, 380),
	})

	require.NoError(t, err)
	assert.Empty(t, f.batchCreator.created)
	require.NotNil(t, f.portioner.calls[0].TargetBatchID)
	assert.Equal(t, "batch-pkg-lot-1", *f.portioner.calls[0].TargetBatchID)
}

func TestPackWeightPortions_checksTheWholeListAgainstTheLotBeforeChangingAnything(t *testing.T) {
	f := newPackFixture()
	f.stock.grams = 4000
	expiry := time.Date(2030, 10, 10, 0, 0, 0, 0, time.UTC)
	f.batches.byID["batch-lot-1"] = invDomain.Batch{ID: "batch-lot-1", SKUID: "sku-bulk-1", ExpiryDate: &expiry}
	lot := "batch-lot-1"

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BulkBatchID: &lot, Lines: lines(1, 430, 10, 380),
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock, "430 + 3800 = 4230g, lot has 4000g")
	assert.Empty(t, f.skuCreator.created)
	assert.Empty(t, f.portioner.calls)
	assert.Equal(t, "sku-bulk-1", f.stock.gotSKU)
	assert.Equal(t, "warehouse-1", f.stock.gotWH)
	assert.Equal(t, &lot, f.stock.gotLot)
}

func TestPackWeightPortions_rejectsBatchBelongingToAnotherSKU(t *testing.T) {
	f := newPackFixture()
	f.batches.byID["batch-other"] = invDomain.Batch{ID: "batch-other", SKUID: "sku-different"}
	lot := "batch-other"

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BulkBatchID: &lot, Lines: lines(1, 404),
	})

	assert.ErrorIs(t, err, invDomain.ErrBatchDoesNotBelongToSKU)
}

func TestPackWeightPortions_rejectsEmptyList(t *testing.T) {
	f := newPackFixture()

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1"})

	assert.ErrorIs(t, err, pmsDomain.ErrNoWeightsProvided)
}

func TestPackWeightPortions_rejectsNonPositiveWeightOrQuantity(t *testing.T) {
	for name, l := range map[string][]apppms.WeightPackageLine{
		"zero weight":   lines(1, 0),
		"zero quantity": lines(0, 380),
		"negative qty":  lines(-2, 380),
	} {
		t.Run(name, func(t *testing.T) {
			f := newPackFixture()
			_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
				BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: l,
			})
			assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
		})
	}
}

func TestPackWeightPortions_rejectsSourceThatIsNotAScaleSoldWeightPool(t *testing.T) {
	for name, mutate := range map[string]func(*pmsDomain.SKU){
		"unit type":     func(s *pmsDomain.SKU) { s.UnitType = pmsDomain.SKUUnitTypeUnit },
		"no scale code": func(s *pmsDomain.SKU) { s.ScaleCode = nil },
		"no kilo price": func(s *pmsDomain.SKU) { s.PricePerKiloMinorUnits = nil },
	} {
		t.Run(name, func(t *testing.T) {
			f := newPackFixture()
			f.setBulk(mutate)
			_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
				BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(1, 404),
			})
			assert.ErrorIs(t, err, pmsDomain.ErrBulkSKUNotWeightPool)
		})
	}
}

func TestPackWeightPortions_namesTheWeightThatFailed(t *testing.T) {
	f := newPackFixture()
	f.portioner.err = errors.New("boom")

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(2, 404),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404g")
}

func TestPackWeightPortions_honorsCustomSKUCodePrefix(t *testing.T) {
	f := newPackFixture()

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(1, 404), SKUCodePrefix: "DRIED-BEEF",
	})

	require.NoError(t, err)
	assert.Equal(t, "DRIED-BEEF-404G", f.skuCreator.created[0].SKUCode)
}

func TestPackWeightPortions_switchesASimpleProductToVariantsModeFirst(t *testing.T) {
	f := newPackFixture()
	categoryID := "cat-1"
	f.catalog.products["product-1"] = pmsDomain.Product{
		ID: "product-1", Name: "Carne Seca", Description: "Dried beef", CategoryID: &categoryID,
		Status: pmsDomain.ProductStatusActive, HasVariants: false, Tags: []string{"meat"},
	}

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(2, 380),
	})

	require.NoError(t, err)
	require.Len(t, f.productUpdater.calls, 1)
	assert.Equal(t, apppms.UpdateProductInput{
		ID: "product-1", Name: "Carne Seca", Description: "Dried beef", CategoryID: &categoryID,
		Status: pmsDomain.ProductStatusActive, HasVariants: true, Tags: []string{"meat"},
	}, f.productUpdater.calls[0], "only HasVariants changes")
	require.Len(t, f.variantCreator.created, 1)
}

func TestPackWeightPortions_leavesASimpleProductAloneWhenOnlyRestocking(t *testing.T) {
	f := newPackFixture()
	p := f.catalog.products["product-1"]
	p.HasVariants = false
	f.catalog.products["product-1"] = p
	f.addPackage("sku-existing-380", 380)

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(2, 380),
	})

	require.NoError(t, err)
	assert.Empty(t, f.productUpdater.calls)
}

func TestPackWeightPortions_newVariantsGoAfterTheExistingOnes(t *testing.T) {
	f := newPackFixture()
	f.addPackage("sku-existing-380", 380)

	_, err := f.useCase(t).Execute(context.Background(), apppms.PackWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Lines: lines(1, 430, 1, 540),
	})

	require.NoError(t, err)
	require.Len(t, f.variantCreator.created, 2)
	assert.Equal(t, 2, f.variantCreator.created[0].Position)
	assert.Equal(t, 3, f.variantCreator.created[1].Position)
}
