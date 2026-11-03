package pms_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateCategory_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewCreateCategoryUseCase(newMockGormDB(t), &stubCategoryRepo{}, outbox)

	_, err := uc.Execute(context.Background(), apppms.CreateCategoryInput{Name: "Sweets"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.category.created", e.Subject)
	assert.Equal(t, "category-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Sweets", "slug": "sweets", "parent_id": nil}, e.Data())
}

func TestDeleteCategory_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubCategoryRepo{findByIDResult: &pmsDomain.Category{ID: "category-1", Name: "Sweets"}}
	uc := apppms.NewDeleteCategoryUseCase(newMockGormDB(t), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "category-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.category.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Sweets"}, e.Data())
}

func TestUploadMedia_RecordsUploaded(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	productID := "product-1"
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 2), &stubMediaProductRepo{product: &pmsDomain.Product{ID: productID}},
		&stubMediaVariantRepo{}, &stubMediaRepo{}, &stubFileStorage{returnedURL: "https://x/a.jpg"}, outbox)

	_, err := uc.Execute(context.Background(), apppms.UploadMediaInput{ProductID: &productID, Filename: "a.jpg", ContentType: "image/jpeg", Data: strings.NewReader("x")})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.media.uploaded", e.Subject)
	assert.Equal(t, "media-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"product_id": "product-1", "variant_id": nil, "url": "https://x/a.jpg", "position": float64(0)}, e.Data())
}

func TestDeleteMedia_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	productID := "product-1"
	repo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "media-1", ProductID: &productID, URL: "u"}}
	uc := apppms.NewDeleteMediaUseCase(newMockGormDBN(t, 1), repo, &stubFileStorage{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "media-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.media.deleted", e.Subject)
	assert.Equal(t, map[string]any{"product_id": "product-1", "variant_id": nil}, e.Data())
}

func TestUpsertSKUCurrencyPrice_RecordsChangedPrices(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSKUCurrencyPriceRepo{byKey: map[string]pmsDomain.SKUCurrencyPrice{
		"sku-1|GBP": {SKUID: "sku-1", CurrencyCode: "GBP", PriceMinorUnits: 900},
	}}
	uc := apppms.NewUpsertSKUCurrencyPriceUseCase(newMockGormDBN(t, 1), &stubSKURepo{}, repo, outbox)

	_, err := uc.Execute(context.Background(), apppms.UpsertSKUCurrencyPriceInput{SKUID: "sku-1", CurrencyCode: "GBP", PriceMinorUnits: 950})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.sku_currency_price.saved", e.Subject)
	assert.Equal(t, "sku-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"currency_code": "GBP"}, e.Data())
	assert.Equal(t, map[string][2]any{"price_minor_units": {float64(900), float64(950)}}, e.Changes())
}

func TestDeleteSKUCurrencyPrice_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewDeleteSKUCurrencyPriceUseCase(newMockGormDBN(t, 1), &stubSKUCurrencyPriceRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "sku-1", "GBP"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.sku_currency_price.deleted", e.Subject)
	assert.Equal(t, "sku-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"currency_code": "GBP"}, e.Data())
}

func TestSetManualCostPrice_RecordsCostPriceChanged(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	before := int64(400)
	repo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &before, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed}}
	uc := apppms.NewSetManualCostPriceUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(context.Background(), "sku-1", 1250)

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.sku.cost_price_changed", e.Subject)
	assert.Equal(t, map[string][2]any{
		"cost_price_minor_units": {float64(400), float64(1250)}, "cost_price_source": {"computed", "manual"},
	}, e.Changes())
	assert.Equal(t, "manual", e.Data()["reason"])
}

func TestClearCostPriceOverride_RecordsCostPriceChanged(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	uc := apppms.NewClearCostPriceOverrideUseCase(newMockGormDB(t), repo, stubLastReceiptCost{}, outbox)

	_, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.sku.cost_price_changed", e.Subject)
	assert.Equal(t, map[string][2]any{"cost_price_source": {"manual", "computed"}}, e.Changes())
	assert.Equal(t, "override_cleared", e.Data()["reason"])
}

func TestApplyReceiptCost_RecordsInventoryCostAndCostPriceChanged(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	cost, avg := int64(450), int64(440)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost, InventoryCostMinorUnits: &avg,
		CostPriceSource: pmsDomain.SKUCostPriceSourceComputed}}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{},
		&stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}, outbox)

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: "purchase_order_line_item", ReferenceID: "line-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, outbox.Events, 2)
	data := map[string]any{"reason": "receipt", "reference_type": "purchase_order_line_item", "reference_id": "line-1"}
	inv, price := outbox.Events[0], outbox.Events[1]
	assert.Equal(t, "pms.sku.inventory_cost_changed", inv.Subject)
	assert.Equal(t, map[string][2]any{"inventory_cost_minor_units": {float64(440), float64(467)}}, inv.Changes())
	assert.Equal(t, data, inv.Data())
	assert.Equal(t, "pms.sku.cost_price_changed", price.Subject)
	assert.Equal(t, map[string][2]any{"cost_price_minor_units": {float64(450), float64(520)}}, price.Changes())
	assert.Equal(t, data, price.Data())
}

func TestCorrectCost_RecordsInventoryCostChanged(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	cost, avg := int64(320), int64(300)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost, InventoryCostMinorUnits: &avg, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed}}
	uc := apppms.NewCorrectCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{},
		&stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 100}}}, outbox)

	_, err := uc.Execute(context.Background(), apppms.CorrectCostInput{SKUID: "sku-1", AdjustmentValueMinorUnits: 5000,
		ReferenceType: "purchase_order_line_item", ReferenceID: "line-1", OccurredAt: time.Now()})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.sku.inventory_cost_changed", e.Subject)
	assert.Equal(t, map[string][2]any{"inventory_cost_minor_units": {float64(300), float64(350)}}, e.Changes())
	assert.Equal(t, map[string]any{"reason": "correction", "reference_type": "purchase_order_line_item", "reference_id": "line-1",
		"adjustment_value_minor_units": float64(5000)}, e.Data(), "the supplier cost history reprices the line from this")
}

func TestBarcodeAlias_RecordsCreatedAndDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	_, err := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), &stubSKURepo{}, &stubBarcodeAliasRepo{}, outbox).
		Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "sku-1", Code: "0630509892495", UnitsPerScan: 12})
	require.NoError(t, err)
	require.NoError(t, apppms.NewDeleteBarcodeAliasUseCase(newMockGormDB(t), &stubBarcodeAliasRepo{}, outbox).Execute(context.Background(), "alias-1"))

	require.Equal(t, []string{"pms.barcode_alias.created", "pms.barcode_alias.deleted"}, outbox.Subjects())
	assert.Equal(t, map[string]any{"sku_id": "sku-1", "code": "0630509892495", "units_per_scan": float64(12)}, outbox.Events[0].Data())
	assert.Equal(t, "alias-1", outbox.Events[1].Payload["entity_id"])
}

func TestScaleCodeAlias_RecordsCreatedAndDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	_, err := apppms.NewCreateScaleCodeAliasUseCase(newMockGormDB(t), &stubSKURepo{}, &stubScaleCodeAliasRepo{}, outbox).
		Execute(context.Background(), apppms.CreateScaleCodeAliasInput{SKUID: "sku-1", Code: "94722"})
	require.NoError(t, err)
	require.NoError(t, apppms.NewDeleteScaleCodeAliasUseCase(newMockGormDB(t), &stubScaleCodeAliasRepo{}, outbox).Execute(context.Background(), "scalealias-1"))

	require.Equal(t, []string{"pms.scale_code_alias.created", "pms.scale_code_alias.deleted"}, outbox.Subjects())
	assert.Equal(t, map[string]any{"sku_id": "sku-1", "code": "94722"}, outbox.Events[0].Data())
}

func TestDuplicateProduct_RecordsDuplicated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	source := &pmsDomain.Product{ID: "source-1", Name: "T-Shirt"}
	productRepo := &stubDupProductRepo{source: source, takenSlugs: map[string]bool{}}
	variantRepo := &stubDupVariantRepo{source: []pmsDomain.Variant{{ID: "variant-1", ProductID: "source-1"}}}
	skuRepo := &stubDupSKURepo{byVariant: map[string][]pmsDomain.SKU{"variant-1": {{ID: "sku-1", SKUCode: "TS"}, {ID: "sku-2", SKUCode: "TS2"}}}, takenCodes: map[string]bool{}}
	uc := apppms.NewDuplicateProductUseCase(newMockGormDB(t), productRepo, variantRepo, skuRepo, outbox)

	_, err := uc.Execute(context.Background(), "source-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "pms.product.duplicated", e.Subject)
	assert.Equal(t, "new-product-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"source_product_id": "source-1", "variant_count": float64(1), "sku_count": float64(2)}, e.Data())
}

func TestCatalogUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()

	_, err := apppms.NewCreateCategoryUseCase(newMockGormDB(t), &stubCategoryRepo{}, failing).Execute(ctx, apppms.CreateCategoryInput{Name: "x"})
	assert.ErrorIs(t, err, boom, "create category")
	assert.ErrorIs(t, apppms.NewDeleteCategoryUseCase(newMockGormDB(t), &stubCategoryRepo{}, failing).Execute(ctx, "c"), boom, "delete category")
	assert.ErrorIs(t, apppms.NewDeleteSKUCurrencyPriceUseCase(newMockGormDBN(t, 1), &stubSKUCurrencyPriceRepo{}, failing).Execute(ctx, "sku-1", "GBP"), boom, "delete price")
	_, err = apppms.NewSetManualCostPriceUseCase(newMockGormDB(t), &stubSKURepo{}, failing).Execute(ctx, "sku-1", 10)
	assert.ErrorIs(t, err, boom, "manual cost")
	assert.ErrorIs(t, apppms.NewDeleteBarcodeAliasUseCase(newMockGormDB(t), &stubBarcodeAliasRepo{}, failing).Execute(ctx, "a"), boom, "delete alias")
}
