package inventory_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestDiscardBatch_recordsALostMovementViaMarkDamagedOrLost(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}
	recorder := &stubDamagedOrLostRecorder{}
	storage := &stubFileStorage{}
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), batchRepo, recorder, storage, &eventrecordtest.Outbox{})

	batchID := "batch-1"
	record, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID, Quantity: 5, Reason: "expired", DiscardedBy: "user-1",
	})

	require.NoError(t, err)
	require.Len(t, recorder.calls, 1)
	require.Equal(t, "sku-1", recorder.calls[0].SKUID)
	require.Equal(t, "wh-1", recorder.calls[0].WarehouseID)
	require.Equal(t, "batch-1", *recorder.calls[0].BatchID)
	require.Equal(t, int64(5), recorder.calls[0].Quantity)
	require.Equal(t, invDomain.MovementTypeLost, recorder.calls[0].Type)
	require.Equal(t, "expired", recorder.calls[0].Reason)
	require.Equal(t, "user-1", recorder.calls[0].DiscardedBy, "passed through to MarkDamagedOrLostUseCase, which writes the DiscardRecord itself")
	require.Nil(t, recorder.calls[0].PhotoURL)

	require.Equal(t, "discard-1", record.ID, "returns exactly the DiscardRecord MarkDamagedOrLostUseCase produced")
	require.Equal(t, []string{"batch-1"}, batchRepo.markReviewed)
}

func TestDiscardBatch_generalLedgerDiscardHasNoBatchToMarkReviewed(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}
	recorder := &stubDamagedOrLostRecorder{}
	storage := &stubFileStorage{}
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), batchRepo, recorder, storage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: nil, Quantity: 3, Reason: "expired",
	})

	require.NoError(t, err)
	require.Nil(t, recorder.calls[0].BatchID)
	require.Empty(t, batchRepo.markReviewed)
}

func TestDiscardBatch_uploadsAnOptionalPhotoBeforeRecordingTheDiscard(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}
	recorder := &stubDamagedOrLostRecorder{}
	storage := &stubFileStorage{saveURL: "https://example.com/photo.jpg"}
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), batchRepo, recorder, storage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 2, Reason: "expired",
		PhotoFilename: "photo.jpg", PhotoContentType: "image/jpeg", PhotoData: strings.NewReader("fake-jpeg-bytes"),
	})

	require.NoError(t, err)
	require.Equal(t, "photo.jpg", storage.savedName)
	require.Equal(t, []byte("fake-jpeg-bytes"), storage.savedData)
	require.NotNil(t, recorder.calls[0].PhotoURL)
	require.Equal(t, "https://example.com/photo.jpg", *recorder.calls[0].PhotoURL)
}

func TestDiscardBatch_rejectsAnUnsupportedPhotoContentType(t *testing.T) {
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}, &stubDamagedOrLostRecorder{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 2, Reason: "expired",
		PhotoFilename: "doc.pdf", PhotoContentType: "application/pdf", PhotoData: strings.NewReader("x"),
	})

	require.ErrorIs(t, err, invDomain.ErrUnsupportedPhotoType)
}

func TestDiscardBatch_rejectsNonPositiveQuantity(t *testing.T) {
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}, &stubDamagedOrLostRecorder{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 0})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestDiscardBatch_propagatesInsufficientStockFromTheMovementRecorder(t *testing.T) {
	recorder := &stubDamagedOrLostRecorder{err: invDomain.ErrInsufficientStock}
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}, recorder, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 100})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}
