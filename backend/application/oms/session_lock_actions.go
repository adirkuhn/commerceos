//events:none releasing a picking/packing session lock is lock bookkeeping (2-minute TTL), not a business change
package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type packingLockReleaser interface {
	FindByID(ctx context.Context, id string) (*omsDomain.PackingSession, error)
	ReleaseLock(ctx context.Context, id string) error
}

type pickingLockReleaser interface {
	FindByID(ctx context.Context, id string) (*omsDomain.PickingSession, error)
	ReleaseLock(ctx context.Context, id string) error
}

// ReleasePackingSessionLockUseCase frees the caller's lock when they leave
// an order (select another, go back, close the tab), so it doesn't sit
// locked until the heartbeat goes stale. A no-op unless the caller holds
// the lock -- leaving never frees someone else's.
type ReleasePackingSessionLockUseCase struct {
	db   *gorm.DB
	repo packingLockReleaser
}

func NewReleasePackingSessionLockUseCase(db *gorm.DB, repo packingLockReleaser) *ReleasePackingSessionLockUseCase {
	return &ReleasePackingSessionLockUseCase{db: db, repo: repo}
}

func (uc *ReleasePackingSessionLockUseCase) Execute(ctx context.Context, sessionID, userID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.repo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if s.LockedBy == nil || *s.LockedBy != userID {
			return nil
		}
		return uc.repo.ReleaseLock(ctx, sessionID)
	})
	if err != nil {
		return fmt.Errorf("releasePackingSessionLock: %w", err)
	}
	return nil
}

// ReleasePickingSessionLockUseCase mirrors ReleasePackingSessionLockUseCase.
type ReleasePickingSessionLockUseCase struct {
	db   *gorm.DB
	repo pickingLockReleaser
}

func NewReleasePickingSessionLockUseCase(db *gorm.DB, repo pickingLockReleaser) *ReleasePickingSessionLockUseCase {
	return &ReleasePickingSessionLockUseCase{db: db, repo: repo}
}

func (uc *ReleasePickingSessionLockUseCase) Execute(ctx context.Context, sessionID, userID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.repo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if s.LockedBy == nil || *s.LockedBy != userID {
			return nil
		}
		return uc.repo.ReleaseLock(ctx, sessionID)
	})
	if err != nil {
		return fmt.Errorf("releasePickingSessionLock: %w", err)
	}
	return nil
}

type packingLockTaker interface {
	FindByID(ctx context.Context, id string) (*omsDomain.PackingSession, error)
	AcquireLock(ctx context.Context, id, userID string, at time.Time) error
}

type pickingLockTaker interface {
	FindByID(ctx context.Context, id string) (*omsDomain.PickingSession, error)
	AcquireLock(ctx context.Context, id, userID string, at time.Time) error
}

// TakeOverPackingSessionUseCase is the explicit "Take over" action: takes
// the lock when it's free, stale, already the caller's, or the caller is an
// admin (see SessionLock.CanTakeOver). The previous holder's next write or
// heartbeat then gets 409.
type TakeOverPackingSessionUseCase struct {
	db   *gorm.DB
	repo packingLockTaker
}

func NewTakeOverPackingSessionUseCase(db *gorm.DB, repo packingLockTaker) *TakeOverPackingSessionUseCase {
	return &TakeOverPackingSessionUseCase{db: db, repo: repo}
}

func (uc *TakeOverPackingSessionUseCase) Execute(ctx context.Context, sessionID string, actor omsDomain.LockActor) (*omsDomain.PackingSession, error) {
	var session *omsDomain.PackingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if session, err = uc.repo.FindByID(ctx, sessionID); err != nil {
			return err
		}
		if session.Status != omsDomain.PackingSessionStatusInProgress {
			return omsDomain.ErrPackingSessionNotActive
		}
		now := time.Now()
		if !session.Lock().CanTakeOver(actor, now) {
			return omsDomain.ErrPackingSessionLocked
		}
		if err := uc.repo.AcquireLock(ctx, sessionID, actor.UserID, now); err != nil {
			return err
		}
		session.ApplyLock(session.Lock().AcquiredBy(actor.UserID, now))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("takeOverPackingSession: %w", err)
	}
	return session, nil
}

// TakeOverPickingSessionUseCase mirrors TakeOverPackingSessionUseCase.
type TakeOverPickingSessionUseCase struct {
	db   *gorm.DB
	repo pickingLockTaker
}

func NewTakeOverPickingSessionUseCase(db *gorm.DB, repo pickingLockTaker) *TakeOverPickingSessionUseCase {
	return &TakeOverPickingSessionUseCase{db: db, repo: repo}
}

func (uc *TakeOverPickingSessionUseCase) Execute(ctx context.Context, sessionID string, actor omsDomain.LockActor) (*omsDomain.PickingSession, error) {
	var session *omsDomain.PickingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if session, err = uc.repo.FindByID(ctx, sessionID); err != nil {
			return err
		}
		if !session.IsOpen() {
			return omsDomain.ErrPickingSessionNotActive
		}
		now := time.Now()
		if !session.Lock().CanTakeOver(actor, now) {
			return omsDomain.ErrPickingSessionLocked
		}
		if err := uc.repo.AcquireLock(ctx, sessionID, actor.UserID, now); err != nil {
			return err
		}
		session.ApplyLock(session.Lock().AcquiredBy(actor.UserID, now))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("takeOverPickingSession: %w", err)
	}
	return session, nil
}
