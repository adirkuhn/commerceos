package inventory_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

type stubDamagedOrLostRecorder struct {
	calls []appinv.MarkDamagedOrLostInput
	err   error
}

func (s *stubDamagedOrLostRecorder) Execute(_ context.Context, in appinv.MarkDamagedOrLostInput) (*invDomain.StockMovement, *invDomain.DiscardRecord, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, nil, s.err
	}
	return &invDomain.StockMovement{ID: "movement-1"}, &invDomain.DiscardRecord{ID: "discard-1", MovementID: "movement-1"}, nil
}

func TestWriteOffExpiredBatch_recordsALostMovementForTheBatchsRemainingQuantityThenMarksReviewed(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{
		{BatchID: strPtr("batch-1"), QuantityDelta: 10},
		{BatchID: strPtr("batch-1"), QuantityDelta: -3},
	}}
	recorder := &stubDamagedOrLostRecorder{}
	uc := appinv.NewWriteOffExpiredBatchUseCase(newMockGormDBN(t, 2), batchRepo, movementRepo, recorder, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1", "user-1")

	require.NoError(t, err)
	require.Len(t, recorder.calls, 1)
	require.Equal(t, "sku-1", recorder.calls[0].SKUID)
	require.Equal(t, "wh-1", recorder.calls[0].WarehouseID)
	require.Equal(t, "batch-1", *recorder.calls[0].BatchID)
	require.Equal(t, int64(7), recorder.calls[0].Quantity)
	require.Equal(t, invDomain.MovementTypeLost, recorder.calls[0].Type)
	require.Equal(t, []string{"batch-1"}, batchRepo.markReviewed)
}

func TestWriteOffExpiredBatch_zeroRemainingQuantitySkipsMovementButStillMarksReviewed(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{
		{BatchID: strPtr("batch-1"), QuantityDelta: 5},
		{BatchID: strPtr("batch-1"), QuantityDelta: -5},
	}}
	recorder := &stubDamagedOrLostRecorder{}
	uc := appinv.NewWriteOffExpiredBatchUseCase(newMockGormDBN(t, 2), batchRepo, movementRepo, recorder, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1", "user-1")

	require.NoError(t, err)
	require.Empty(t, recorder.calls, "nothing left to write off -- no movement should be created")
	require.Equal(t, []string{"batch-1"}, batchRepo.markReviewed)
}

func TestWriteOffExpiredBatch_propagatesFindError(t *testing.T) {
	batchRepo := &stubBatchRepo{findErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewWriteOffExpiredBatchUseCase(newMockGormDB(t), batchRepo, &stubStockMovementRepo{}, &stubDamagedOrLostRecorder{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1", "user-1")

	require.Error(t, err)
}

func TestWriteOffExpiredBatch_propagatesRecorderError(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{{BatchID: strPtr("batch-1"), QuantityDelta: 4}}}
	recorder := &stubDamagedOrLostRecorder{err: invDomain.ErrInvalidQuantity}
	uc := appinv.NewWriteOffExpiredBatchUseCase(newMockGormDB(t), batchRepo, movementRepo, recorder, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1", "user-1")

	require.Error(t, err)
	require.Empty(t, batchRepo.markReviewed, "must not mark reviewed if the write-off movement failed")
}
