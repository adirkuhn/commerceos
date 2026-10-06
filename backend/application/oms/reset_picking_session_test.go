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

func TestResetPickingSession_deletesAllScansAndReleasesLock(t *testing.T) {
	db := newMockGormDB(t)
	user := "user-1"
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &user,
	}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ID: "scan-1"}, {ID: "scan-2"}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewResetPickingSessionUseCase(db, sessionRepo, scanRepo, outbox)

	session, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"scan-1", "scan-2"}, scanRepo.deleted)
	assert.Equal(t, []string{"session-1"}, sessionRepo.released)
	require.Len(t, sessionRepo.updated, 1)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, sessionRepo.updated[0].Status)
	assert.Nil(t, sessionRepo.updated[0].LockedBy)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)

	assert.Contains(t, outbox.Subjects(), "oms.picking_session.reset")

}

func TestResetPickingSession_allowsFromPaused(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusPaused,
	}}
	scanRepo := &stubPickScanRepo{}
	uc := appoms.NewResetPickingSessionUseCase(db, sessionRepo, scanRepo, &eventrecordtest.Outbox{})

	session, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	assert.Equal(t, omsDomain.PickingSessionStatusInProgress, session.Status)
}

func TestResetPickingSession_rejectsAlreadyCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusCompleted,
	}}
	scanRepo := &stubPickScanRepo{}
	uc := appoms.NewResetPickingSessionUseCase(db, sessionRepo, scanRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "session-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionAlreadyCompleted)
	assert.Empty(t, scanRepo.deleted)
	assert.Empty(t, sessionRepo.released)
}

func TestResetPickingSession_propagatesSessionNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDErr: omsDomain.ErrPickingSessionNotFound}
	uc := appoms.NewResetPickingSessionUseCase(db, sessionRepo, &stubPickScanRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}
