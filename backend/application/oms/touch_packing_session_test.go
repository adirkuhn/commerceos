package oms_test

import (
	"context"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTouchPackingSession_refreshesOwnLock(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusInProgress}}
	uc := appoms.NewTouchPackingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestTouchPackingSession_rejectsWhenTakenOverByAnotherFreshUser(t *testing.T) {
	db := newMockGormDB(t)
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{
		ID: "session-1", Status: omsDomain.PackingSessionStatusInProgress, LockedBy: &otherUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewTouchPackingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
	assert.Empty(t, sessionRepo.acquiredLocks)
}

func TestTouchPackingSession_rejectsCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}}
	uc := appoms.NewTouchPackingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}
