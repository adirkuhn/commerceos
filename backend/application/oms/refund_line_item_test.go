package oms_test

import (
	"context"
	"testing"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundLineItem_rejectsZeroOrNegativeQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewRefundLineItemUseCase(db, &stubPackingSessionRepo{}, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubGroupPermissionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: 0})
	assert.ErrorIs(t, err, omsDomain.ErrInvalidRefundQuantity)

	_, err = uc.Execute(context.Background(), appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: -1})
	assert.ErrorIs(t, err, omsDomain.ErrInvalidRefundQuantity)
}

func TestRefundLineItem_rejectsQuantityExceedingLineItemQuantity(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 2}}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, &stubGroupPermissionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: 3})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidRefundQuantity)
}

func TestRefundLineItem_customerUserWithoutPermissionIsDenied(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 2}}
	permRepo := &stubGroupPermissionRepo{hasPermissionResult: false}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, permRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", Quantity: 1, RequesterUserID: "user-1", RequesterRole: "customer_user",
	})

	assert.ErrorIs(t, err, omsDomain.ErrPermissionDenied)
	assert.Equal(t, "user-1", permRepo.gotUserID)
	assert.Equal(t, "oms.line_item.refund", permRepo.gotRule.String())
}

func TestRefundLineItem_customerUserWithPermissionIsAllowed(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 2}}
	permRepo := &stubGroupPermissionRepo{hasPermissionResult: true}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, permRepo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", Quantity: 1, RequesterUserID: "user-1", RequesterRole: "customer_user", ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	assert.Equal(t, omsDomain.PackScanActionRefund, result.Scan.ActionType)
}

// customer_admin/customer_owner/super_admin bypass the fine-grained check
// entirely -- HasPermission must never even be called for them.
func TestRefundLineItem_adminAndOwnerBypassPermissionCheck(t *testing.T) {
	for _, role := range []string{"customer_admin", "customer_owner", "super_admin", ""} {
		db := newMockGormDB(t)
		sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
		scanRepo := &stubPackScanRepo{}
		lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 2}}
		permRepo := &stubGroupPermissionRepo{hasPermissionResult: false}
		uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, permRepo, &eventrecordtest.Outbox{})

		_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{
			SessionID: "session-1", LineItemID: "li-1", Quantity: 1, RequesterRole: role,
		})

		require.NoError(t, err, "role %q must bypass the permission check", role)
		assert.Equal(t, "", permRepo.gotUserID, "role %q must never call HasPermission", role)
	}
}

func TestRefundLineItem_rejectsInactiveSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusCompleted}}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, &stubPackScanRepo{}, &stubSalesOrderLineItemRepo{}, &stubGroupPermissionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}

func TestRefundLineItem_rejectsLineItemFromAnotherOrder(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-other", SKUID: strPtr("sku-1"), Quantity: 2}}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, &stubGroupPermissionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrLineItemNotOnOrder)
}

func TestRefundLineItem_rejectsLineItemWithNoLinkedSKU(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: nil, Quantity: 2}}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, &stubPackScanRepo{}, lineItemRepo, &stubGroupPermissionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: 1})

	assert.ErrorIs(t, err, omsDomain.ErrRefundRequiresLinkedSKU)
}

func TestRefundLineItem_happyPathCreatesRefundScan(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 3}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewRefundLineItemUseCase(db, sessionRepo, scanRepo, lineItemRepo, &stubGroupPermissionRepo{}, outbox)

	result, err := uc.Execute(context.Background(), appoms.RefundLineItemInput{
		SessionID: "session-1", LineItemID: "li-1", Quantity: 2, Reason: "damaged", ScannedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, scanRepo.created, 1)
	created := scanRepo.created[0]
	assert.Equal(t, omsDomain.PackScanActionRefund, created.ActionType)
	require.NotNil(t, created.LineItemID)
	assert.Equal(t, "li-1", *created.LineItemID)
	assert.Equal(t, "sku-1", created.SKUID)
	assert.Equal(t, int64(2), created.Quantity)
	assert.Equal(t, "damaged", created.Reason)
	assert.Equal(t, "user-1", created.ScannedBy)
	assert.Equal(t, omsDomain.PackScanActionRefund, result.Scan.ActionType)

	assert.Contains(t, outbox.Subjects(), "oms.pack_scan.created")
	assert.Equal(t, "refund", outbox.Events[0].Data()["action_type"])

}
