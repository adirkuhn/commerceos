package oms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPackingQueue_listsAwaitingOrdersWithTheirLocks(t *testing.T) {
	now := time.Now()
	since := now.Add(-5 * time.Minute)
	picker, packer := "picker", "packer"
	salesOrderRepo := &stubSalesOrderRepo{listResult: []omsDomain.SalesOrder{{ID: "so-1"}, {ID: "so-2"}, {ID: "so-3"}}, listTotal: 3}
	pickingRepo := &stubPickingSessionRepo{openBySalesOrders: []omsDomain.PickingSession{
		{SalesOrderID: "so-1", LockedBy: &picker, LockedAt: &now, LockedSince: &since},
	}}
	packingRepo := &stubPackingSessionRepo{activeBySalesOrders: []omsDomain.PackingSession{
		{SalesOrderID: "so-2", LockedBy: &packer, LockedAt: &now, LockedSince: &since},
	}}
	uc := appoms.NewListPackingQueueUseCase(newMockGormDB(t), salesOrderRepo, pickingRepo, packingRepo)

	result, err := uc.Execute(context.Background(), 2, 3)

	require.NoError(t, err)
	assert.True(t, salesOrderRepo.gotListFilter.AwaitingPacking)
	assert.Equal(t, 3, salesOrderRepo.gotListFilter.Limit)
	assert.Equal(t, 3, salesOrderRepo.gotListFilter.Offset)
	assert.Equal(t, []string{"so-1", "so-2", "so-3"}, pickingRepo.gotOpenIDs)
	assert.Equal(t, []string{"so-1", "so-2", "so-3"}, packingRepo.gotActiveIDs)
	assert.Equal(t, int64(3), result.Total)
	require.Len(t, result.Entries, 3)
	require.NotNil(t, result.Entries[0].Lock)
	assert.Equal(t, omsDomain.QueueLockPicking, result.Entries[0].Lock.Activity)
	require.NotNil(t, result.Entries[1].Lock)
	assert.Equal(t, "packer", result.Entries[1].Lock.UserID)
	assert.Nil(t, result.Entries[2].Lock)
}

func TestListPackingQueue_defaultsAndCapsPaging(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewListPackingQueueUseCase(newMockGormDB(t), salesOrderRepo, &stubPickingSessionRepo{}, &stubPackingSessionRepo{})

	_, err := uc.Execute(context.Background(), 0, 100000)

	require.NoError(t, err)
	assert.Equal(t, 200, salesOrderRepo.gotListFilter.Limit)
	assert.Equal(t, 0, salesOrderRepo.gotListFilter.Offset)
}

func TestListPackingQueue_skipsSessionLookupsForEmptyPage(t *testing.T) {
	pickingRepo := &stubPickingSessionRepo{}
	uc := appoms.NewListPackingQueueUseCase(newMockGormDB(t), &stubSalesOrderRepo{}, pickingRepo, &stubPackingSessionRepo{})

	result, err := uc.Execute(context.Background(), 1, 20)

	require.NoError(t, err)
	assert.Empty(t, result.Entries)
	assert.Nil(t, pickingRepo.gotOpenIDs)
}

func TestListPackingQueue_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	orders := []omsDomain.SalesOrder{{ID: "so-1"}}

	_, err := appoms.NewListPackingQueueUseCase(newMockGormDB(t), &stubSalesOrderRepo{listErr: boom}, &stubPickingSessionRepo{}, &stubPackingSessionRepo{}).
		Execute(context.Background(), 1, 20)
	assert.ErrorIs(t, err, boom)

	_, err = appoms.NewListPackingQueueUseCase(newMockGormDB(t), &stubSalesOrderRepo{listResult: orders}, &stubPickingSessionRepo{openErr: boom}, &stubPackingSessionRepo{}).
		Execute(context.Background(), 1, 20)
	assert.ErrorIs(t, err, boom)

	_, err = appoms.NewListPackingQueueUseCase(newMockGormDB(t), &stubSalesOrderRepo{listResult: orders}, &stubPickingSessionRepo{}, &stubPackingSessionRepo{listActiveErr: boom}).
		Execute(context.Background(), 1, 20)
	assert.ErrorIs(t, err, boom)
}
