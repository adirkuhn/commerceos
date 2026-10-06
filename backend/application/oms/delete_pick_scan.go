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

// DeletePickScanUseCase removes a staged PickScan row before it's ever
// reached real inventory/order state -- the scanner app's fix for a
// mis-scan. Only legal while the session is still in_progress -- once
// Finish runs, every row is either real or gone, so there's nothing left
// here to undo (see CompletePickingUseCase's own doc comment -- replay is
// the only place a staged row becomes real). Mirrors
// procurement.DeleteBookingScanUseCase.
type DeletePickScanUseCase struct {
	outbox      eventrecord.Enqueuer
	db          *gorm.DB
	sessionRepo omsDomain.PickingSessionRepository
	scanRepo    omsDomain.PickScanRepository
}

func NewDeletePickScanUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	scanRepo omsDomain.PickScanRepository,
	outbox eventrecord.Enqueuer,
) *DeletePickScanUseCase {
	return &DeletePickScanUseCase{outbox: outbox, db: db, sessionRepo: sessionRepo, scanRepo: scanRepo}
}

func (uc *DeletePickScanUseCase) Execute(ctx context.Context, sessionID, scanID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PickingSessionStatusInProgress {
			return omsDomain.ErrPickingSessionNotActive
		}

		scans, err := uc.scanRepo.ListBySession(ctx, sessionID)
		if err != nil {
			return err
		}
		var found *omsDomain.PickScan
		for i := range scans {
			if scans[i].ID == scanID {
				found = &scans[i]
				break
			}
		}
		if found == nil {
			return omsDomain.ErrPickScanNotFound
		}

		if err := uc.scanRepo.Delete(ctx, scanID); err != nil {
			return err
		}
		return recordPickScan(ctx, uc.outbox, events.PickScanDeleted, *found)
	})
	if err != nil {
		return fmt.Errorf("deletePickScan: %w", err)
	}
	return nil
}
