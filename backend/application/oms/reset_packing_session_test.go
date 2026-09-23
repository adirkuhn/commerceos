package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetPackingSession_deletesAllScansAndReleasesLock(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusInProgress}}
	scanRepo := &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1"}, {ID: "scan-2"}}}
	uc := appoms.NewResetPackingSessionUseCase(db, sessionRepo, scanRepo)

	session, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"scan-1", "scan-2"}, scanRepo.deleted)
	assert.Equal(t, []string{"session-1"}, sessionRepo.released)
	assert.Nil(t, session.LockedBy)
}

func TestResetPackingSession_rejectsCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}}
	scanRepo := &stubPackScanRepo{}
	uc := appoms.NewResetPackingSessionUseCase(db, sessionRepo, scanRepo)

	_, err := uc.Execute(context.Background(), "session-1")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionAlreadyCompleted)
	assert.Empty(t, scanRepo.deleted)
}
