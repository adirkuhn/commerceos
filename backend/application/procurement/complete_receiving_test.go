package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteReceiving_marksRecordApplied(t *testing.T) {
	db := newMockGormDB(t)
	recRepo := &stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{
		ID: "receiving-record-1", Status: procurementDomain.ReceivingRecordStatusPendingReview,
	}}
	uc := appprocurement.NewCompleteReceivingUseCase(db, recRepo)

	rec, err := uc.Execute(context.Background(), "receiving-record-1")

	require.NoError(t, err)
	require.Len(t, recRepo.updated, 1)
	assert.Equal(t, procurementDomain.ReceivingRecordStatusApplied, recRepo.updated[0].Status)
	assert.Equal(t, procurementDomain.ReceivingRecordStatusApplied, rec.Status)
}

func TestCompleteReceiving_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	recRepo := &stubReceivingRecordRepo{findErr: procurementDomain.ErrReceivingRecordNotFound}
	uc := appprocurement.NewCompleteReceivingUseCase(db, recRepo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrReceivingRecordNotFound)
}

func TestCompleteReceiving_propagatesUpdateError(t *testing.T) {
	db := newMockGormDB(t)
	recRepo := &stubReceivingRecordRepo{
		findResult: &procurementDomain.ReceivingRecord{ID: "receiving-record-1"},
		updateErr:  assert.AnError,
	}
	uc := appprocurement.NewCompleteReceivingUseCase(db, recRepo)

	_, err := uc.Execute(context.Background(), "receiving-record-1")

	require.Error(t, err)
}
