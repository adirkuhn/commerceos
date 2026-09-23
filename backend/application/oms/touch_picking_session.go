package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// TouchPickingSessionUseCase refreshes the caller's picking-session lock --
// the scanner/admin app calls this on a heartbeat interval while a session
// stays open, so PickingLockTTL never expires under a picker who's still
// actively working the order. If the lock has been taken over by someone
// else in the meantime (this caller's own lock went stale and another
// picker grabbed it), it returns ErrPickingSessionLocked so the frontend
// can tell the picker they lost the session rather than silently keep
// scanning into an order someone else now owns.
type TouchPickingSessionUseCase struct {
	db          *gorm.DB
	sessionRepo omsDomain.PickingSessionRepository
}

func NewTouchPickingSessionUseCase(db *gorm.DB, sessionRepo omsDomain.PickingSessionRepository) *TouchPickingSessionUseCase {
	return &TouchPickingSessionUseCase{db: db, sessionRepo: sessionRepo}
}

func (uc *TouchPickingSessionUseCase) Execute(ctx context.Context, sessionID, userID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if !session.IsOpen() {
			return omsDomain.ErrPickingSessionNotActive
		}
		// A heartbeat never takes a lock by force, even for an admin -- that
		// would silently undo someone else's takeover.
		now := time.Now()
		if session.Lock().IsHeldByOther(userID, now) {
			return omsDomain.ErrPickingSessionLocked
		}
		return uc.sessionRepo.AcquireLock(ctx, sessionID, userID, now)
	})
	if err != nil {
		return fmt.Errorf("touchPickingSession: %w", err)
	}
	return nil
}
