package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// TouchPackingSessionUseCase mirrors TouchPickingSessionUseCase -- see that
// type's doc comment.
type TouchPackingSessionUseCase struct {
	db          *gorm.DB
	sessionRepo omsDomain.PackingSessionRepository
}

func NewTouchPackingSessionUseCase(db *gorm.DB, sessionRepo omsDomain.PackingSessionRepository) *TouchPackingSessionUseCase {
	return &TouchPackingSessionUseCase{db: db, sessionRepo: sessionRepo}
}

func (uc *TouchPackingSessionUseCase) Execute(ctx context.Context, sessionID, userID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.Status == omsDomain.PackingSessionStatusCompleted {
			return omsDomain.ErrPackingSessionNotActive
		}
		if session.LockedBy != nil && *session.LockedBy != userID &&
			session.LockedAt != nil && time.Since(*session.LockedAt) < omsDomain.PackingLockTTL {
			return omsDomain.ErrPackingSessionLocked
		}
		return uc.sessionRepo.AcquireLock(ctx, sessionID, userID, time.Now())
	})
	if err != nil {
		return fmt.Errorf("touchPackingSession: %w", err)
	}
	return nil
}
