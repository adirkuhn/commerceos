package oms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ResetPickingSessionUseCase discards every staged scan on a session and
// hands it back fresh -- the picker's "start over" escape hatch when
// they've scanned into the wrong box, want to switch boxes mid-pick, or
// just want a clean slate. Only legal while the session is in_progress or
// paused: once CompletePickingUseCase's replay has run, staged rows are
// either real (stock moved, line items mutated) or gone, so there's
// nothing left here to undo -- same one-way-door rule DeletePickScanUseCase
// already enforces for a single scan, applied to the whole session.
type ResetPickingSessionUseCase struct {
	outbox      eventrecord.Enqueuer
	db          *gorm.DB
	sessionRepo omsDomain.PickingSessionRepository
	scanRepo    omsDomain.PickScanRepository
}

func NewResetPickingSessionUseCase(db *gorm.DB, sessionRepo omsDomain.PickingSessionRepository, scanRepo omsDomain.PickScanRepository, outbox eventrecord.Enqueuer) *ResetPickingSessionUseCase {
	return &ResetPickingSessionUseCase{outbox: outbox, db: db, sessionRepo: sessionRepo, scanRepo: scanRepo}
}

func (uc *ResetPickingSessionUseCase) Execute(ctx context.Context, sessionID string) (*omsDomain.PickingSession, error) {
	var session omsDomain.PickingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if s.Status == omsDomain.PickingSessionStatusCompleted {
			return omsDomain.ErrPickingSessionAlreadyCompleted
		}

		scans, err := uc.scanRepo.ListBySession(ctx, sessionID)
		if err != nil {
			return err
		}
		for _, scan := range scans {
			if err := uc.scanRepo.Delete(ctx, scan.ID); err != nil {
				return err
			}
		}

		// Release the lock explicitly (map-based) rather than relying on
		// Update below -- see PickingSessionRepository.AcquireLock's doc
		// for why Update can't clear a pointer field to NULL.
		if err := uc.sessionRepo.ReleaseLock(ctx, sessionID); err != nil {
			return err
		}

		s.Status = omsDomain.PickingSessionStatusInProgress
		s.PausedAt = nil
		s.LockedBy = nil
		s.LockedAt = nil
		if err := uc.sessionRepo.Update(ctx, s); err != nil {
			return err
		}
		session = *s
		return recordSession(ctx, uc.outbox, events.PickingSessionReset, s.ID, s.SalesOrderID, eventrecord.Data{"removed_scan_count": len(scans)})
	})
	if err != nil {
		return nil, fmt.Errorf("resetPickingSession: %w", err)
	}
	return &session, nil
}
