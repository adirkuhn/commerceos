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

// DeletePackScanUseCase mirrors DeletePickScanUseCase -- removes one pack
// scan (fix a mis-scan, or move an item to a different box) while the
// session is still in_progress.
type DeletePackScanUseCase struct {
	outbox      eventrecord.Enqueuer
	db          *gorm.DB
	sessionRepo omsDomain.PackingSessionRepository
	scanRepo    omsDomain.PackScanRepository
}

func NewDeletePackScanUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	scanRepo omsDomain.PackScanRepository,
	outbox eventrecord.Enqueuer,
) *DeletePackScanUseCase {
	return &DeletePackScanUseCase{outbox: outbox, db: db, sessionRepo: sessionRepo, scanRepo: scanRepo}
}

func (uc *DeletePackScanUseCase) Execute(ctx context.Context, sessionID, scanID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PackingSessionStatusInProgress {
			return omsDomain.ErrPackingSessionNotActive
		}

		scans, err := uc.scanRepo.ListBySession(ctx, sessionID)
		if err != nil {
			return err
		}
		var found *omsDomain.PackScan
		for i := range scans {
			if scans[i].ID == scanID {
				found = &scans[i]
				break
			}
		}
		if found == nil {
			return omsDomain.ErrPackScanNotFound
		}

		if err := uc.scanRepo.Delete(ctx, scanID); err != nil {
			return err
		}
		return recordPackScan(ctx, uc.outbox, events.PackScanDeleted, *found)
	})
	if err != nil {
		return fmt.Errorf("deletePackScan: %w", err)
	}
	return nil
}
