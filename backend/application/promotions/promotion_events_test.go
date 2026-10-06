package promotions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

var promoStart = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func TestCreatePromotion_RecordsCreatedWithLines(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, outbox)

	_, err := uc.Execute(context.Background(), apppromotions.CreatePromotionInput{
		Name: "Xmas", Type: promotionsDomain.PromotionTypePercentage, StartsAt: promoStart,
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}},
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "promotions.promotion.created", e.Subject)
	assert.Equal(t, "promo-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"name": "Xmas", "type": "percentage", "status": "draft", "starts_at": "2026-11-01T00:00:00Z", "ends_at": nil,
		"lines": []any{map[string]any{"sku_id": "sku-1", "batch_id": nil, "discount_type": "percentage", "discount_value": float64(1000)}},
	}, e.Data())
}

func TestUpdatePromotion_RecordsChangedFieldsAndLines(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{
		ID: "promo-1", Name: "Xmas", Type: promotionsDomain.PromotionTypePercentage, StartsAt: promoStart, Status: promotionsDomain.PromotionStatusDraft,
	}}
	lines := &stubPromotionSKURepo{listResult: []promotionsDomain.PromotionSKU{
		{ID: "l-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
	}}
	uc := apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), promoRepo, lines, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &stubLineReverter{}, outbox)

	_, err := uc.Execute(context.Background(), apppromotions.UpdatePromotionInput{
		ID: "promo-1", Name: "Xmas", Type: promotionsDomain.PromotionTypePercentage, StartsAt: promoStart, Status: promotionsDomain.PromotionStatusActive,
		Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1500}},
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "promotions.promotion.updated", e.Subject)
	changes := e.Changes()
	assert.Equal(t, [2]any{"draft", "active"}, changes["status"])
	require.Contains(t, changes, "lines")
	assert.NotContains(t, changes, "name")
}

func TestDeletePromotion_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Name: "Xmas"}}
	uc := apppromotions.NewDeletePromotionUseCase(newMockGormDBN(t, 2), promoRepo, &stubPromotionSKURepo{}, &stubLineReverter{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "promo-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "promotions.promotion.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Xmas"}, e.Data())
}

func TestApplyPromotionSKU_RecordsApplied(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	prev := int64(950)
	skuLookup := &stubSKUExistenceChecker{findResult: &pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 1000}}
	uc := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 2), skuLookup, &stubSetPromotionSalePrice{applyPrevPrice: &prev}, &stubPromotionSKURepo{}, outbox)

	err := uc.Execute(context.Background(), promotionsDomain.PromotionSKU{
		ID: "line-1", PromotionID: "promo-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "promotions.promotion_line.applied", e.Subject)
	assert.Equal(t, "line-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"promotion_id": "promo-1", "sku_id": "sku-1", "sale_price_minor_units": float64(900), "previous_sale_price_minor_units": float64(950),
	}, e.Data())
}

func TestRevertPromotionSKU_RecordsReverted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	prev := int64(950)
	uc := apppromotions.NewRevertPromotionSKUUseCase(newMockGormDB(t), &stubSetPromotionSalePrice{}, &stubPromotionSKURepo{}, outbox)

	err := uc.Execute(context.Background(), promotionsDomain.PromotionSKU{ID: "line-1", PromotionID: "promo-1", SKUID: "sku-1", PreviousSalePriceMinorUnits: &prev})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "promotions.promotion_line.reverted", e.Subject)
	assert.Equal(t, map[string]any{"promotion_id": "promo-1", "sku_id": "sku-1", "restored_sale_price_minor_units": float64(950)}, e.Data())
}

// The last clearance line finishing ends the promotion: both recorded.
func TestFinishClearanceLine_RecordsFinishedAndPromotionEnded(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	promoRepo := &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Status: promotionsDomain.PromotionStatusActive}}
	uc := apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{}, &stubClearanceLineRepo{unfinished: 0}, promoRepo, &stubLineReverter{}, outbox)

	_, err := uc.ExecuteForLine(context.Background(), clearanceLine(false))

	require.NoError(t, err)
	require.Equal(t, []string{"promotions.promotion_line.finished", "promotions.promotion.updated"}, outbox.Subjects())
	assert.Equal(t, map[string]any{"promotion_id": "promo-1", "sku_id": "sku-1", "batch_id": "batch-1"}, outbox.Events[0].Data())
	assert.Equal(t, map[string][2]any{"status": {"active", "ended"}}, outbox.Events[1].Changes())
	assert.Equal(t, map[string]any{"reason": "clearance_finished"}, outbox.Events[1].Data())
}

func TestPromotionUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()
	found := func() *stubPromotionRepo {
		return &stubPromotionRepo{findResult: &promotionsDomain.Promotion{ID: "promo-1", Status: promotionsDomain.PromotionStatusActive}}
	}

	_, err := apppromotions.NewCreatePromotionUseCase(newMockGormDB(t), &stubPromotionRepo{}, &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, failing).
		Execute(ctx, apppromotions.CreatePromotionInput{Name: "X", Type: promotionsDomain.PromotionTypePercentage, StartsAt: promoStart, Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}}})
	assert.ErrorIs(t, err, boom, "create")
	_, err = apppromotions.NewUpdatePromotionUseCase(newMockGormDBN(t, 2), found(), &stubPromotionSKURepo{}, &stubSKUExistenceChecker{}, &stubSKUBatchLister{}, &stubLineReverter{}, failing).
		Execute(ctx, apppromotions.UpdatePromotionInput{ID: "promo-1", Name: "X", Type: promotionsDomain.PromotionTypePercentage, StartsAt: promoStart, Status: promotionsDomain.PromotionStatusDraft, Lines: []apppromotions.PromotionLineInput{{SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000}}})
	assert.ErrorIs(t, err, boom, "update")
	assert.ErrorIs(t, apppromotions.NewDeletePromotionUseCase(newMockGormDBN(t, 2), found(), &stubPromotionSKURepo{}, &stubLineReverter{}, failing).Execute(ctx, "promo-1"), boom, "delete")
	assert.ErrorIs(t, apppromotions.NewRevertPromotionSKUUseCase(newMockGormDB(t), &stubSetPromotionSalePrice{}, &stubPromotionSKURepo{}, failing).
		Execute(ctx, promotionsDomain.PromotionSKU{ID: "line-1"}), boom, "revert")
	_, err = apppromotions.NewFinishClearanceLinesUseCase(newMockGormDB(t), &stubSKUBatchLister{}, &stubClearanceLineRepo{unfinished: 1}, found(), &stubLineReverter{}, failing).
		ExecuteForLine(ctx, clearanceLine(false))
	assert.ErrorIs(t, err, boom, "finish")
}
