package oms_test

import (
	"context"
	"testing"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLinkSKURepo struct {
	result *pmsDomain.SKU
	err    error
}

func (s *stubLinkSKURepo) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	if s.result == nil && s.err == nil {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return s.result, s.err
}

func TestLinkLineItemSKU_linksAndClearsNeedsSKUReviewWhenNoneRemainUnlinked(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) { o.NeedsSKUReview = true })}
	lineItemRepo := &stubSalesOrderLineItemRepo{
		findByIDResult:   &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "sales-order-1", SKUID: nil},
		listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "sales-order-1", SKUID: strPtr("sku-1")}},
	}
	skuRepo := &stubLinkSKURepo{result: &pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewLinkLineItemSKUUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, outbox)

	order, err := uc.Execute(context.Background(), appoms.LinkLineItemSKUInput{
		SalesOrderID: "sales-order-1", LineItemID: "li-1", SKUID: "sku-1",
	})

	require.NoError(t, err)
	require.NotNil(t, order)
	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, "li-1", lineItemRepo.updateCalls[0].ID)
	assert.Equal(t, "sku-1", lineItemRepo.updateCalls[0].Fields["sku_id"])
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	assert.Equal(t, false, salesOrderRepo.updateCalls[0].Fields["needs_sku_review"], "no other line item is still unlinked")

	assert.Contains(t, outbox.Subjects(), "oms.sales_order_line_item.linked")
	assert.Contains(t, outbox.Subjects(), "oms.sales_order.updated")

}

func TestLinkLineItemSKU_keepsNeedsSKUReviewTrueWhenAnotherLineIsStillUnlinked(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	lineItemRepo := &stubSalesOrderLineItemRepo{
		findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "sales-order-1", SKUID: nil},
		listBySalesOrder: []omsDomain.SalesOrderLineItem{
			{ID: "li-1", SalesOrderID: "sales-order-1", SKUID: strPtr("sku-1")},
			{ID: "li-2", SalesOrderID: "sales-order-1", SKUID: nil},
		},
	}
	skuRepo := &stubLinkSKURepo{result: &pmsDomain.SKU{ID: "sku-1"}}
	uc := appoms.NewLinkLineItemSKUUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.LinkLineItemSKUInput{
		SalesOrderID: "sales-order-1", LineItemID: "li-1", SKUID: "sku-1",
	})

	require.NoError(t, err)
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	assert.Equal(t, true, salesOrderRepo.updateCalls[0].Fields["needs_sku_review"], "li-2 is still unlinked")
}

func TestLinkLineItemSKU_rejectsALineItemFromADifferentOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{
		findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "some-other-order", SKUID: nil},
	}
	skuRepo := &stubLinkSKURepo{result: &pmsDomain.SKU{ID: "sku-1"}}
	uc := appoms.NewLinkLineItemSKUUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.LinkLineItemSKUInput{
		SalesOrderID: "sales-order-1", LineItemID: "li-1", SKUID: "sku-1",
	})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemNotOnOrder)
	assert.Empty(t, lineItemRepo.updateCalls, "must not link before the order-ownership check passes")
}

func TestLinkLineItemSKU_rejectsAnUnknownSKUID(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{
		findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "sales-order-1", SKUID: nil},
	}
	skuRepo := &stubLinkSKURepo{err: pmsDomain.ErrSKUNotFound}
	uc := appoms.NewLinkLineItemSKUUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.LinkLineItemSKUInput{
		SalesOrderID: "sales-order-1", LineItemID: "li-1", SKUID: "missing-sku",
	})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
	assert.Empty(t, lineItemRepo.updateCalls)
}
