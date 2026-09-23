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

func TestTouchPickingSession_refreshesOwnLock(t *testing.T) {
	db := newMockGormDB(t)
	user := "user-1"
	lockedAt := time.Now().Add(-1 * time.Minute)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &user, LockedAt: &lockedAt,
	}}
	uc := appoms.NewTouchPickingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestTouchPickingSession_claimsUnlockedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress,
	}}
	uc := appoms.NewTouchPickingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, sessionRepo.acquiredLocks)
}

func TestTouchPickingSession_rejectsWhenTakenOverByAnotherFreshUser(t *testing.T) {
	db := newMockGormDB(t)
	otherUser := "user-2"
	lockedAt := time.Now()
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &otherUser, LockedAt: &lockedAt,
	}}
	uc := appoms.NewTouchPickingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionLocked)
	assert.Empty(t, sessionRepo.acquiredLocks)
}

func TestTouchPickingSession_rejectsCompletedSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{
		ID: "session-1", Status: omsDomain.PickingSessionStatusCompleted,
	}}
	uc := appoms.NewTouchPickingSessionUseCase(db, sessionRepo)

	err := uc.Execute(context.Background(), "session-1", "user-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}
