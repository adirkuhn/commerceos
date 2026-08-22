package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PausePickingUseCase is the scanner app's "Pause" action -- "Continue"
// doesn't need its own use case, since re-calling
// StartOrResumePickingUseCase for the same order already flips a paused
// session back to in_progress and hands the same session back. Also how
// "switch to a different order mid-pick" works: pause this session, then
// start/resume the other order's.
type PausePickingUseCase struct {
	db          *gorm.DB
	sessionRepo omsDomain.PickingSessionRepository
}

func NewPausePickingUseCase(db *gorm.DB, sessionRepo omsDomain.PickingSessionRepository) *PausePickingUseCase {
	return &PausePickingUseCase{db: db, sessionRepo: sessionRepo}
}

func (uc *PausePickingUseCase) Execute(ctx context.Context, id string) (*omsDomain.PickingSession, error) {
	var session *omsDomain.PickingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		session, err = uc.sessionRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PickingSessionStatusInProgress {
			return omsDomain.ErrPickingSessionNotActive
		}
		now := time.Now()
		session.Status = omsDomain.PickingSessionStatusPaused
		session.PausedAt = &now
		return uc.sessionRepo.Update(ctx, session)
	})
	if err != nil {
		return nil, fmt.Errorf("pausePicking: %w", err)
	}
	return session, nil
}
