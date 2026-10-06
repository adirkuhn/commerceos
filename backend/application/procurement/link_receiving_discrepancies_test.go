package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLinkFixtureRepo() *stubReceivingDiscrepancyRepo {
	return &stubReceivingDiscrepancyRepo{byID: map[string]*procurementDomain.ReceivingDiscrepancy{
		"unexpected-1": {ID: "unexpected-1", ReceivingRecordID: "record-1", Type: procurementDomain.DiscrepancyTypeUnexpectedItem, Status: procurementDomain.DiscrepancyStatusOpen},
		"missing-1":    {ID: "missing-1", ReceivingRecordID: "record-1", Type: procurementDomain.DiscrepancyTypeMissingItem, Status: procurementDomain.DiscrepancyStatusOpen},
	}}
}

func TestLinkReceivingDiscrepancies_happyPath(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "unexpected-1", "missing-1")

	require.NoError(t, err)
	require.Len(t, repo.updated, 2)
	assert.Equal(t, procurementDomain.DiscrepancyStatusAcknowledged, repo.byID["unexpected-1"].Status)
	assert.Equal(t, procurementDomain.DiscrepancyStatusAcknowledged, repo.byID["missing-1"].Status)
}

func TestLinkReceivingDiscrepancies_mismatchedReceivingRecord(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	repo.byID["missing-1"].ReceivingRecordID = "record-2"
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "unexpected-1", "missing-1")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyMismatch)
	assert.Empty(t, repo.updated)
}

func TestLinkReceivingDiscrepancies_wrongTypeOnUnexpectedSide(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	repo.byID["unexpected-1"].Type = procurementDomain.DiscrepancyTypeQuantityMismatch
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "unexpected-1", "missing-1")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyMismatch)
}

func TestLinkReceivingDiscrepancies_wrongTypeOnMissingSide(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	repo.byID["missing-1"].Type = procurementDomain.DiscrepancyTypeExpiryMismatch
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "unexpected-1", "missing-1")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyMismatch)
}

func TestLinkReceivingDiscrepancies_alreadyAcknowledged(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	repo.byID["missing-1"].Status = procurementDomain.DiscrepancyStatusAcknowledged
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "unexpected-1", "missing-1")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyAlreadyAcknowledged)
}

func TestLinkReceivingDiscrepancies_unexpectedNotFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	repo.findErr = procurementDomain.ErrDiscrepancyNotFound
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "nonexistent", "missing-1")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyNotFound)
}

func TestLinkReceivingDiscrepancies_missingSideNotFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	repo := newLinkFixtureRepo()
	repo.findErr = procurementDomain.ErrDiscrepancyNotFound
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "unexpected-1", "nonexistent")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyNotFound)
}
