package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type fakeBatchFinder struct {
	byID map[string]invDomain.Batch
}

func (f *fakeBatchFinder) FindByID(_ context.Context, id string) (*invDomain.Batch, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, invDomain.ErrBatchNotFound
	}
	return &b, nil
}

type fakeSKUCreator struct {
	created []apppms.CreateSKUInput
	err     error
	nextID  int
}

func (f *fakeSKUCreator) Execute(_ context.Context, in apppms.CreateSKUInput) (*pmsDomain.SKU, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.nextID++
	f.created = append(f.created, in)
	sku := pmsDomain.SKU{
		ID: sequentialID(f.nextID), VariantID: in.VariantID, SKUCode: in.SKUCode, Barcode: in.Barcode,
		PriceMinorUnits: in.PriceMinorUnits, CurrencyCode: in.CurrencyCode, WeightGrams: in.WeightGrams,
		UnitType: in.UnitType, PortionedFromSKUID: in.PortionedFromSKUID, PricingMode: in.PricingMode, Status: in.Status,
	}
	return &sku, nil
}

func sequentialID(n int) string {
	return "sku-portion-" + string(rune('0'+n))
}

type fakeBatchCreator struct {
	created []invapp.CreateBatchInput
	err     error
	nextID  int
}

func (f *fakeBatchCreator) Execute(_ context.Context, in invapp.CreateBatchInput) (*invDomain.Batch, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.nextID++
	f.created = append(f.created, in)
	return &invDomain.Batch{ID: sequentialID(f.nextID), SKUID: in.SKUID, WarehouseID: in.WarehouseID, LotNumber: in.LotNumber, ExpiryDate: in.ExpiryDate}, nil
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

func weightPoolBulkSKU() pmsDomain.SKU {
	scaleCode := "2994721"
	pricePerKilo := int64(29990) // EUR 29.99/kg in minor units
	return pmsDomain.SKU{
		ID: "sku-bulk-1", VariantID: "variant-1", SKUCode: "CARNE-SECA", CurrencyCode: "EUR",
		UnitType: pmsDomain.SKUUnitTypeKilo, PricingMode: pmsDomain.SKUPricingModePerWeight,
		PricePerKiloMinorUnits: &pricePerKilo, ScaleCode: &scaleCode,
	}
}

func TestBulkAddWeightPortions_createsOnePortionedSKUPerWeight(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: ptrSKU(weightPoolBulkSKU())}
	skuCreator := &fakeSKUCreator{}
	batchCreator := &fakeBatchCreator{}
	portioner := &fakePortioner{}
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, &fakeBatchFinder{}, skuCreator, batchCreator, portioner)

	result, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404, 458},
	})

	require.NoError(t, err)
	require.Len(t, result.Created, 2)
	require.Len(t, skuCreator.created, 2)
	require.Len(t, portioner.calls, 2)
	assert.Empty(t, batchCreator.created, "no BulkBatchID given -- no expiry to carry forward, no new batch")

	first := skuCreator.created[0]
	assert.Equal(t, "CARNE-SECA-404G", first.SKUCode)
	assert.Equal(t, pmsDomain.SKUUnitTypeUnit, first.UnitType)
	assert.Equal(t, pmsDomain.SKUPricingModeFixed, first.PricingMode)
	require.NotNil(t, first.PortionedFromSKUID)
	assert.Equal(t, "sku-bulk-1", *first.PortionedFromSKUID)
	require.NotNil(t, first.WeightGrams)
	assert.Equal(t, int64(404), *first.WeightGrams)
	// 29990 minor units/kg * 404g / 1000 = 12115.96 -> rounds to 12116
	// (round-half-up), matching lineTotalForScannedWeight's existing
	// rounding exactly -- EUR 121.16 for 404g at EUR 29.99/kg.
	assert.Equal(t, int64(12116), first.PriceMinorUnits)
	// Barcode must be a real, decodable weight-embedded barcode for 404g.
	decoded, err := pmsDomain.DecodeWeightEmbeddedBarcode(first.Barcode)
	require.NoError(t, err)
	assert.Equal(t, "2994721", decoded.ScaleCode)
	assert.Equal(t, int64(404), decoded.WeightGrams)

	assert.Equal(t, int64(458), *portioner.calls[1].WeightGrams)
	assert.Nil(t, portioner.calls[0].TargetBatchID)
}

func TestBulkAddWeightPortions_copiesExpiryFromTheChosenBulkBatch(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: ptrSKU(weightPoolBulkSKU())}
	expiry := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	batchFinder := &fakeBatchFinder{byID: map[string]invDomain.Batch{
		"batch-lot-1": {ID: "batch-lot-1", SKUID: "sku-bulk-1", LotNumber: "LOT-1", ExpiryDate: &expiry},
	}}
	batchCreator := &fakeBatchCreator{}
	portioner := &fakePortioner{}
	batchID := "batch-lot-1"
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, batchFinder, &fakeSKUCreator{}, batchCreator, portioner)

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BulkBatchID: &batchID, WeightsGrams: []int64{404},
	})

	require.NoError(t, err)
	require.Len(t, batchCreator.created, 1)
	assert.Equal(t, "LOT-1", batchCreator.created[0].LotNumber)
	assert.Equal(t, &expiry, batchCreator.created[0].ExpiryDate)
	require.Len(t, portioner.calls, 1)
	require.NotNil(t, portioner.calls[0].TargetBatchID)
	assert.Equal(t, "sku-portion-1", *portioner.calls[0].TargetBatchID)
	require.NotNil(t, portioner.calls[0].BatchID)
	assert.Equal(t, "batch-lot-1", *portioner.calls[0].BatchID, "draws down the BULK sku's own batch, not the new one")
}

func TestBulkAddWeightPortions_rejectsBatchBelongingToAnotherSKU(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: ptrSKU(weightPoolBulkSKU())}
	batchFinder := &fakeBatchFinder{byID: map[string]invDomain.Batch{
		"batch-other": {ID: "batch-other", SKUID: "sku-different"},
	}}
	batchID := "batch-other"
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, batchFinder, &fakeSKUCreator{}, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BulkBatchID: &batchID, WeightsGrams: []int64{404},
	})

	assert.ErrorIs(t, err, invDomain.ErrBatchDoesNotBelongToSKU)
}

func TestBulkAddWeightPortions_rejectsEmptyWeightsList(t *testing.T) {
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDBN(t, 0), &stubSKURepo{}, &fakeBatchFinder{}, &fakeSKUCreator{}, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1"})

	assert.ErrorIs(t, err, pmsDomain.ErrNoWeightsProvided)
}

func TestBulkAddWeightPortions_rejectsNonPositiveWeight(t *testing.T) {
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDBN(t, 0), &stubSKURepo{}, &fakeBatchFinder{}, &fakeSKUCreator{}, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404, 0},
	})

	assert.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestBulkAddWeightPortions_rejectsSourceSKUNotAWeightPool(t *testing.T) {
	notAPool := weightPoolBulkSKU()
	notAPool.UnitType = pmsDomain.SKUUnitTypeUnit
	skuRepo := &stubSKURepo{findByIDResult: &notAPool}
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, &fakeBatchFinder{}, &fakeSKUCreator{}, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404},
	})

	assert.ErrorIs(t, err, pmsDomain.ErrBulkSKUNotWeightPool)
}

func TestBulkAddWeightPortions_rejectsSourceSKUWithNoScaleCode(t *testing.T) {
	noScaleCode := weightPoolBulkSKU()
	noScaleCode.ScaleCode = nil
	skuRepo := &stubSKURepo{findByIDResult: &noScaleCode}
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, &fakeBatchFinder{}, &fakeSKUCreator{}, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404},
	})

	assert.ErrorIs(t, err, pmsDomain.ErrBulkSKUNotWeightPool)
}

func TestBulkAddWeightPortions_stopsAndNamesTheFailingWeight(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: ptrSKU(weightPoolBulkSKU())}
	skuCreator := &fakeSKUCreator{err: errors.New("duplicate sku code")}
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, &fakeBatchFinder{}, skuCreator, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404g")
}

func TestBulkAddWeightPortions_defaultsSKUCodePrefixToBulkSKUCode(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: ptrSKU(weightPoolBulkSKU())}
	skuCreator := &fakeSKUCreator{}
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, &fakeBatchFinder{}, skuCreator, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404},
	})

	require.NoError(t, err)
	assert.Equal(t, "CARNE-SECA-404G", skuCreator.created[0].SKUCode)
}

func TestBulkAddWeightPortions_honorsCustomSKUCodePrefix(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: ptrSKU(weightPoolBulkSKU())}
	skuCreator := &fakeSKUCreator{}
	uc := apppms.NewBulkAddWeightPortionsUseCase(newMockGormDB(t), skuRepo, &fakeBatchFinder{}, skuCreator, &fakeBatchCreator{}, &fakePortioner{})

	_, err := uc.Execute(context.Background(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: "sku-bulk-1", WarehouseID: "warehouse-1", WeightsGrams: []int64{404}, SKUCodePrefix: "DRIED-BEEF",
	})

	require.NoError(t, err)
	assert.Equal(t, "DRIED-BEEF-404G", skuCreator.created[0].SKUCode)
}

func ptrSKU(s pmsDomain.SKU) *pmsDomain.SKU { return &s }
