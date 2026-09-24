package oms

import (
	"context"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// Opening a picking/packing session only views it -- the lock is taken by
// the first write (scan, change, finish), so browsing several orders never
// locks them. These claim helpers are that "first write" step: refused
// while someone else actively holds the lock, otherwise take (or refresh)
// it for userID.

type packingLockAcquirer interface {
	AcquireLock(ctx context.Context, id, userID string, at time.Time) error
}

type pickingLockAcquirer interface {
	AcquireLock(ctx context.Context, id, userID string, at time.Time) error
}

func claimPackingLock(ctx context.Context, repo packingLockAcquirer, s *omsDomain.PackingSession, userID string, now time.Time) error {
	if s.Lock().IsHeldByOther(userID, now) {
		return omsDomain.ErrPackingSessionLocked
	}
	if err := repo.AcquireLock(ctx, s.ID, userID, now); err != nil {
		return err
	}
	s.ApplyLock(s.Lock().AcquiredBy(userID, now))
	return nil
}

func claimPickingLock(ctx context.Context, repo pickingLockAcquirer, s *omsDomain.PickingSession, userID string, now time.Time) error {
	if s.Lock().IsHeldByOther(userID, now) {
		return omsDomain.ErrPickingSessionLocked
	}
	if err := repo.AcquireLock(ctx, s.ID, userID, now); err != nil {
		return err
	}
	s.ApplyLock(s.Lock().AcquiredBy(userID, now))
	return nil
}
