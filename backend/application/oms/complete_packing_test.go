package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletePacking_completesAndMarksOrderPacked(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2, Name: "Widget"}}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{LineItemID: "li-1", SKUID: skuID, Quantity: 2}}}
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewCompletePackingUseCase(db, sessionRepo, salesOrderRepo, lineItemRepo, scanRepo)

	result, err := uc.Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	assert.Empty(t, result.MissingItems)
	assert.Equal(t, omsDomain.PackingSessionStatusCompleted, result.Session.Status)
	require.Len(t, sessionRepo.updated, 1)
	assert.Equal(t, "user-1", sessionRepo.updated[0].CompletedBy)
	assert.Equal(t, []string{"session-1"}, sessionRepo.released)
	assert.Equal(t, "order-1", salesOrderRepo.gotUpdateID)
	assert.Equal(t, map[string]any{"packed": true, "fulfilled": true}, salesOrderRepo.gotUpdateFields,
		"packing also marks the order fulfilled -- picking is optional")
}

func TestCompletePacking_reportsMissingItemsButStillCompletes(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 2, Name: "Widget"}}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{LineItemID: "li-1", SKUID: skuID, Quantity: 1}}}
	uc := appoms.NewCompletePackingUseCase(db, sessionRepo, &stubSalesOrderRepo{}, lineItemRepo, scanRepo)

	result, err := uc.Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, result.MissingItems, 1)
	assert.Equal(t, int64(1), result.MissingItems[0].PackedQty)
	assert.Equal(t, int64(2), result.MissingItems[0].ExpectedQty)
	assert.Equal(t, omsDomain.PackingSessionStatusCompleted, result.Session.Status)
}

func TestCompletePacking_rejectsAlreadyCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}}
	uc := appoms.NewCompletePackingUseCase(db, sessionRepo, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubPackScanRepo{})

	_, err := uc.Execute(context.Background(), appoms.CompletePackingInput{SessionID: "session-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}
