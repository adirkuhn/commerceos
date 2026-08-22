package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcknowledgeReceivingDiscrepancy_happyPath(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := &stubReceivingDiscrepancyRepo{findResult: &procurementDomain.ReceivingDiscrepancy{
		ID: "discrepancy-1", ReceivingRecordID: "receiving-record-1",
		Type: procurementDomain.DiscrepancyTypeQuantityMismatch, Status: procurementDomain.DiscrepancyStatusOpen,
	}}
	uc := appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(db, repo)

	d, err := uc.Execute(context.Background(), "discrepancy-1")

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.DiscrepancyStatusAcknowledged, d.Status)
	require.NotNil(t, d.AcknowledgedAt)
	require.Len(t, repo.updated, 1)
	assert.Equal(t, procurementDomain.DiscrepancyStatusAcknowledged, repo.updated[0].Status)
}

func TestAcknowledgeReceivingDiscrepancy_notFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := &stubReceivingDiscrepancyRepo{findErr: procurementDomain.ErrDiscrepancyNotFound}
	uc := appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(db, repo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyNotFound)
}

func TestAcknowledgeReceivingDiscrepancy_alreadyAcknowledged(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := &stubReceivingDiscrepancyRepo{findResult: &procurementDomain.ReceivingDiscrepancy{
		ID: "discrepancy-1", Status: procurementDomain.DiscrepancyStatusAcknowledged,
	}}
	uc := appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(db, repo)

	_, err := uc.Execute(context.Background(), "discrepancy-1")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyAlreadyAcknowledged)
	assert.Empty(t, repo.updated)
}
