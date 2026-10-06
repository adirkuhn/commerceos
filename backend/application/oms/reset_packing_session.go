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

// ResetPackingSessionUseCase mirrors ResetPickingSessionUseCase -- discards
// every pack scan on a session and releases its lock, the packer's
// "start over" escape hatch for switching boxes or fixing a mis-scan spree.
// Only legal while the session is in_progress -- 422 once completed.
type ResetPackingSessionUseCase struct {
	outbox      eventrecord.Enqueuer
	db          *gorm.DB
	sessionRepo omsDomain.PackingSessionRepository
	scanRepo    omsDomain.PackScanRepository
}

func NewResetPackingSessionUseCase(db *gorm.DB, sessionRepo omsDomain.PackingSessionRepository, scanRepo omsDomain.PackScanRepository, outbox eventrecord.Enqueuer) *ResetPackingSessionUseCase {
	return &ResetPackingSessionUseCase{outbox: outbox, db: db, sessionRepo: sessionRepo, scanRepo: scanRepo}
}

func (uc *ResetPackingSessionUseCase) Execute(ctx context.Context, sessionID string) (*omsDomain.PackingSession, error) {
	var session omsDomain.PackingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if s.Status == omsDomain.PackingSessionStatusCompleted {
			return omsDomain.ErrPackingSessionAlreadyCompleted
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

		if err := uc.sessionRepo.ReleaseLock(ctx, sessionID); err != nil {
			return err
		}

		s.LockedBy = nil
		s.LockedAt = nil
		if err := uc.sessionRepo.Update(ctx, s); err != nil {
			return err
		}
		session = *s
		return recordSession(ctx, uc.outbox, events.PackingSessionReset, s.ID, s.SalesOrderID, eventrecord.Data{"removed_scan_count": len(scans)})
	})
	if err != nil {
		return nil, fmt.Errorf("resetPackingSession: %w", err)
	}
	return &session, nil
}
