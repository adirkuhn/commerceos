package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeletePackScanUseCase mirrors DeletePickScanUseCase -- removes one pack
// scan (fix a mis-scan, or move an item to a different box) while the
// session is still in_progress.
type DeletePackScanUseCase struct {
	db          *gorm.DB
	sessionRepo omsDomain.PackingSessionRepository
	scanRepo    omsDomain.PackScanRepository
}

func NewDeletePackScanUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	scanRepo omsDomain.PackScanRepository,
) *DeletePackScanUseCase {
	return &DeletePackScanUseCase{db: db, sessionRepo: sessionRepo, scanRepo: scanRepo}
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
		found := false
		for _, s := range scans {
			if s.ID == scanID {
				found = true
				break
			}
		}
		if !found {
			return omsDomain.ErrPackScanNotFound
		}

		return uc.scanRepo.Delete(ctx, scanID)
	})
	if err != nil {
		return fmt.Errorf("deletePackScan: %w", err)
	}
	return nil
}
