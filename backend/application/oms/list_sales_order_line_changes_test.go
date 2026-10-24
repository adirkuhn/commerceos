package oms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubAppliedChanges struct {
	out []omsDomain.StagedChange
	err error
}

func (s *stubAppliedChanges) ListAppliedBySalesOrder(_ context.Context, _ string) ([]omsDomain.StagedChange, error) {
	return s.out, s.err
}

type stubLineChangeLines struct{ out []omsDomain.SalesOrderLineItem }

func (s *stubLineChangeLines) ListBySalesOrder(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
	return s.out, nil
}

type stubLineChangeSKUs struct{ byID map[string]pmsDomain.SKU }

func (s *stubLineChangeSKUs) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	sku, ok := s.byID[id]
	if !ok {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return &sku, nil
}

type stubLineChangeVariants struct{}

func (stubLineChangeVariants) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	return &pmsDomain.Variant{ID: id, ProductID: "prod-" + id, Name: "Default"}, nil
}

type stubLineChangeProducts struct{}

func (stubLineChangeProducts) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	return &pmsDomain.Product{ID: id, Name: "Product " + id}, nil
}

func newLineChangesUC(t *testing.T, packing, picking *stubAppliedChanges, lines []omsDomain.SalesOrderLineItem) *appoms.ListSalesOrderLineChangesUseCase {
	skus := &stubLineChangeSKUs{byID: map[string]pmsDomain.SKU{
		"sku-black": {ID: "sku-black", VariantID: "var-black", SKUCode: "FEP-1"},
		"sku-gift":  {ID: "sku-gift", VariantID: "var-gift", SKUCode: "GUA-1"},
	}}
	return appoms.NewListSalesOrderLineChangesUseCase(newMockGormDB(t), appoms.LineChangeSources{Packing: packing, Picking: picking},
		&stubLineChangeLines{out: lines}, skus, stubLineChangeVariants{}, stubLineChangeProducts{})
}

func TestListSalesOrderLineChanges_describesEachChangeWithItsProduct(t *testing.T) {
	covered := int64(4)
	at := time.Date(2026, 10, 24, 10, 49, 0, 0, time.UTC)
	packing := &stubAppliedChanges{out: []omsDomain.StagedChange{
		{ID: "c-sub", Action: omsDomain.ChangeSubstitute, LineItemID: strPtr("line-carioca"), SKUID: "sku-black", Quantity: 4,
			OriginalQtyCovered: &covered, ReturnToStock: true, Reason: "out of stock", ScannedAt: at, ScannedBy: "user-1"},
		{ID: "c-gift", Action: omsDomain.ChangeFreebie, SKUID: "sku-gift", Quantity: 1, Reason: "Guaraná", ScannedAt: at.Add(time.Minute)},
	}}
	lines := []omsDomain.SalesOrderLineItem{
		{ID: "line-carioca", SKUID: strPtr("sku-carioca"), Name: "Feijão Carioca", Quantity: 10},
		{ID: "line-gift", SKUID: strPtr("sku-gift"), Name: "Guaraná", Quantity: 1},
	}

	got, err := newLineChangesUC(t, packing, &stubAppliedChanges{}, lines).Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, got, 2)
	sub := got[0]
	assert.Equal(t, omsDomain.ChangeSubstitute, sub.Action)
	assert.Equal(t, appoms.OrderChangeStepPacking, sub.Step)
	assert.Equal(t, "line-carioca", *sub.LineItemID)
	assert.Equal(t, "FEP-1", sub.SKUCode)
	assert.Equal(t, "Product prod-var-black", sub.ProductName)
	assert.Equal(t, int64(4), *sub.OriginalQtyCovered)
	assert.True(t, sub.ReturnToStock)
	assert.Equal(t, "out of stock", sub.Reason)
	assert.Equal(t, "user-1", sub.By)
	assert.Equal(t, at, sub.At)

	gift := got[1]
	assert.Equal(t, omsDomain.ChangeFreebie, gift.Action)
	require.NotNil(t, gift.LineItemID, "an extra points at the line it became")
	assert.Equal(t, "line-gift", *gift.LineItemID)
}

func TestListSalesOrderLineChanges_mergesPickingAndPackingOldestFirst(t *testing.T) {
	at := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	packing := &stubAppliedChanges{out: []omsDomain.StagedChange{
		{ID: "pack-refund", Action: omsDomain.ChangeRefund, LineItemID: strPtr("line-1"), SKUID: "sku-black", Quantity: 1, ScannedAt: at.Add(time.Hour)},
	}}
	picking := &stubAppliedChanges{out: []omsDomain.StagedChange{
		{ID: "pick-refund", Action: omsDomain.ChangeRefund, LineItemID: strPtr("line-1"), SKUID: "sku-black", Quantity: 1, ScannedAt: at},
	}}

	got, err := newLineChangesUC(t, packing, picking, nil).Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "pick-refund", got[0].ID)
	assert.Equal(t, appoms.OrderChangeStepPicking, got[0].Step)
	assert.Equal(t, "pack-refund", got[1].ID)
}

func TestListSalesOrderLineChanges_keepsAChangeWhoseSKUIsGone(t *testing.T) {
	packing := &stubAppliedChanges{out: []omsDomain.StagedChange{
		{ID: "c1", Action: omsDomain.ChangeSubstitute, LineItemID: strPtr("line-1"), SKUID: "sku-deleted", Quantity: 1},
	}}

	got, err := newLineChangesUC(t, packing, &stubAppliedChanges{}, nil).Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Empty(t, got[0].ProductName)
	assert.Empty(t, got[0].SKUCode)
}

func TestListSalesOrderLineChanges_wrapsAReadError(t *testing.T) {
	packing := &stubAppliedChanges{err: errors.New("db down")}

	_, err := newLineChangesUC(t, packing, &stubAppliedChanges{}, nil).Execute(context.Background(), "so-1")

	assert.ErrorContains(t, err, "listSalesOrderLineChanges")
}
