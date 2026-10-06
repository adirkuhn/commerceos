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

func TestDeletePackScan_removesStagedScan(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1"}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewDeletePackScanUseCase(db, sessionRepo, scanRepo, outbox)

	err := uc.Execute(context.Background(), "session-1", "scan-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"scan-1"}, scanRepo.deleted)

	assert.Contains(t, outbox.Subjects(), "oms.pack_scan.deleted")

}

func TestDeletePackScan_rejectsUnknownScan(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1"}}}
	uc := appoms.NewDeletePackScanUseCase(db, sessionRepo, scanRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "session-1", "missing")

	assert.ErrorIs(t, err, omsDomain.ErrPackScanNotFound)
	assert.Empty(t, scanRepo.deleted)
}

func TestDeletePackScan_rejectsCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1"}}}
	uc := appoms.NewDeletePackScanUseCase(db, sessionRepo, scanRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "session-1", "scan-1")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
	assert.Empty(t, scanRepo.deleted)
}
