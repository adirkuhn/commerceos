package oms_test

import (
	"context"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const weighedLineGrams int64 = 350

func weightPoolSKU(id string, costPerGram *int64) *pmsDomain.SKU {
	return &pmsDomain.SKU{ID: id, UnitType: pmsDomain.SKUUnitTypeKilo, CostPriceMinorUnits: costPerGram}
}

func weighedLine(skuID string, quantity int64) appoms.ReceivePOSSalesOrderLineItemInput {
	grams := weighedLineGrams
	return appoms.ReceivePOSSalesOrderLineItemInput{
		SKUID: strPtr(skuID), Name: "Carne seca", Quantity: quantity, WeightGrams: &grams,
		PriceMinorUnits: 1225, SubtotalMinorUnits: 1225, TotalMinorUnits: 1225,
	}
}

func receiveWeighedOrder(t *testing.T, skuRepo *stubSKURepo, lineItemRepo *stubSalesOrderLineItemRepo, recordMovement *stubRecordMovement, line appoms.ReceivePOSSalesOrderLineItemInput) error {
	t.Helper()
	db := newMockGormDBN(t, 3) // order create, link, freezeCostAtSale
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubOrigins{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, skuRepo, recordMovement, &eventrecordtest.Outbox{})
	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "EUR", OccurredAt: time.Now(),
		TotalMinorUnits: 1225, LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{line},
	})
	return err
}

func TestReceivePOSSalesOrder_storesTheWeightOfAWeighedLine(t *testing.T) {
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-pool": weightPoolSKU("sku-pool", nil)}}

	err := receiveWeighedOrder(t, skuRepo, lineItemRepo, &stubRecordMovement{}, weighedLine("sku-pool", 1))

	require.NoError(t, err)
	require.NotNil(t, lineItemRepo.created[0].WeightGrams)
	assert.Equal(t, weighedLineGrams, *lineItemRepo.created[0].WeightGrams)
}

// A weight pool's stock is counted in grams, so a 350 g sale must take 350
// off it -- not the line's one-piece quantity.
func TestReceivePOSSalesOrder_decrementsAWeightPoolByTheLineWeight(t *testing.T) {
	recordMovement := &stubRecordMovement{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-pool": weightPoolSKU("sku-pool", nil)}}

	err := receiveWeighedOrder(t, skuRepo, &stubSalesOrderLineItemRepo{}, recordMovement, weighedLine("sku-pool", 1))

	require.NoError(t, err)
	assert.Equal(t, weighedLineGrams, recordMovement.calls[0].Quantity)
}

// A per-weight package SKU (UnitType unit) is stocked by package count, so
// its weight is recorded but one package is decremented.
func TestReceivePOSSalesOrder_decrementsAUnitSKUByQuantityEvenWhenWeighed(t *testing.T) {
	recordMovement := &stubRecordMovement{}

	err := receiveWeighedOrder(t, &stubSKURepo{}, &stubSalesOrderLineItemRepo{}, recordMovement, weighedLine("sku-package", 1))

	require.NoError(t, err)
	assert.Equal(t, int64(1), recordMovement.calls[0].Quantity)
}

// A weight pool's cost is per gram, so COGS scales with the weight sold.
func TestReceivePOSSalesOrder_freezesWeightPoolCOGSByTheLineWeight(t *testing.T) {
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	costPerGram := int64(2)
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-pool": weightPoolSKU("sku-pool", &costPerGram)}}

	err := receiveWeighedOrder(t, skuRepo, lineItemRepo, &stubRecordMovement{}, weighedLine("sku-pool", 1))

	require.NoError(t, err)
	assert.Equal(t, costPerGram*weighedLineGrams, lineItemRepo.gotUpdateFields["cogs_minor_units"])
}

// One weighed line is one weighed package: a weight with any other
// quantity would leave it unclear whether the weight is per unit or total.
func TestReceivePOSSalesOrder_rejectsAWeighedLineWithQuantityOtherThanOne(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubOrigins{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "EUR", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{weighedLine("sku-pool", 2)},
	})

	assert.ErrorIs(t, err, omsDomain.ErrWeighedLineItemQuantityMustBeOne)
}

// COGS is what the stock actually cost (the moving average), never the
// cost price -- which is the last receipt's replacement cost (COM-11).
func TestReceivePOSSalesOrder_freezesCOGSFromTheInventoryCostNotTheCostPrice(t *testing.T) {
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	lastReceipt, average := int64(3), int64(2)
	sku := weightPoolSKU("sku-pool", &lastReceipt)
	sku.InventoryCostMinorUnits = &average
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-pool": sku}}

	err := receiveWeighedOrder(t, skuRepo, lineItemRepo, &stubRecordMovement{}, weighedLine("sku-pool", 1))

	require.NoError(t, err)
	assert.Equal(t, average, lineItemRepo.gotUpdateFields["cost_at_sale_minor_units"])
	assert.Equal(t, average*weighedLineGrams, lineItemRepo.gotUpdateFields["cogs_minor_units"])
}
