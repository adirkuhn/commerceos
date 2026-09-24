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

func packSession(holder string, heartbeat time.Time) *omsDomain.PackingSession {
	s := &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}
	if holder != "" {
		s.LockedBy, s.LockedAt, s.LockedSince = &holder, &heartbeat, &heartbeat
	}
	return s
}

func pickSession(holder string, heartbeat time.Time) *omsDomain.PickingSession {
	s := &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}
	if holder != "" {
		s.LockedBy, s.LockedAt, s.LockedSince = &holder, &heartbeat, &heartbeat
	}
	return s
}

func TestReleasePackingSessionLock(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name     string
		session  *omsDomain.PackingSession
		released bool
	}{
		{"holder releases", packSession("user-1", now), true},
		{"someone else's lock stays", packSession("user-2", now), false},
		{"nothing to release", packSession("", now), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubPackingSessionRepo{findByIDResult: tc.session}
			err := appoms.NewReleasePackingSessionLockUseCase(newMockGormDB(t), repo).Execute(context.Background(), "session-1", "user-1")

			require.NoError(t, err)
			if tc.released {
				assert.Equal(t, []string{"session-1"}, repo.released)
			} else {
				assert.Empty(t, repo.released)
			}
		})
	}
}

func TestReleasePickingSessionLock(t *testing.T) {
	repo := &stubPickingSessionRepo{findByIDResult: pickSession("user-1", time.Now())}

	err := appoms.NewReleasePickingSessionLockUseCase(newMockGormDB(t), repo).Execute(context.Background(), "session-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"session-1"}, repo.released)

	other := &stubPickingSessionRepo{findByIDResult: pickSession("user-2", time.Now())}
	require.NoError(t, appoms.NewReleasePickingSessionLockUseCase(newMockGormDB(t), other).Execute(context.Background(), "session-1", "user-1"))
	assert.Empty(t, other.released)
}

func TestTakeOverPackingSession(t *testing.T) {
	now := time.Now()
	stale := now.Add(-omsDomain.SessionLockTTL - time.Minute)

	t.Run("admin takes an active lock", func(t *testing.T) {
		repo := &stubPackingSessionRepo{findByIDResult: packSession("user-2", now)}
		s, err := appoms.NewTakeOverPackingSessionUseCase(newMockGormDB(t), repo).Execute(context.Background(), "session-1", omsDomain.LockActor{UserID: "admin-1", IsAdmin: true})

		require.NoError(t, err)
		assert.Equal(t, []string{"session-1"}, repo.acquiredLocks)
		assert.Equal(t, "admin-1", *s.LockedBy)
		assert.WithinDuration(t, time.Now(), *s.LockedSince, time.Second)
	})

	t.Run("anyone takes a stale lock", func(t *testing.T) {
		repo := &stubPackingSessionRepo{findByIDResult: packSession("user-2", stale)}
		_, err := appoms.NewTakeOverPackingSessionUseCase(newMockGormDB(t), repo).Execute(context.Background(), "session-1", omsDomain.LockActor{UserID: "user-1"})

		require.NoError(t, err)
		assert.Equal(t, []string{"session-1"}, repo.acquiredLocks)
	})

	t.Run("a regular user can't take an active lock", func(t *testing.T) {
		repo := &stubPackingSessionRepo{findByIDResult: packSession("user-2", now)}
		_, err := appoms.NewTakeOverPackingSessionUseCase(newMockGormDB(t), repo).Execute(context.Background(), "session-1", omsDomain.LockActor{UserID: "user-1"})

		assert.ErrorIs(t, err, omsDomain.ErrPackingSessionLocked)
		assert.Empty(t, repo.acquiredLocks)
	})

	t.Run("a completed session can't be taken", func(t *testing.T) {
		s := packSession("", now)
		s.Status = omsDomain.PackingSessionStatusCompleted
		_, err := appoms.NewTakeOverPackingSessionUseCase(newMockGormDB(t), &stubPackingSessionRepo{findByIDResult: s}).Execute(context.Background(), "session-1", omsDomain.LockActor{UserID: "user-1", IsAdmin: true})

		assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
	})
}

func TestTakeOverPickingSession(t *testing.T) {
	repo := &stubPickingSessionRepo{findByIDResult: pickSession("user-2", time.Now())}

	s, err := appoms.NewTakeOverPickingSessionUseCase(newMockGormDB(t), repo).Execute(context.Background(), "session-1", omsDomain.LockActor{UserID: "admin-1", IsAdmin: true})

	require.NoError(t, err)
	assert.Equal(t, "admin-1", *s.LockedBy)

	blocked := &stubPickingSessionRepo{findByIDResult: pickSession("user-2", time.Now())}
	_, err = appoms.NewTakeOverPickingSessionUseCase(newMockGormDB(t), blocked).Execute(context.Background(), "session-1", omsDomain.LockActor{UserID: "user-1"})
	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionLocked)
}
