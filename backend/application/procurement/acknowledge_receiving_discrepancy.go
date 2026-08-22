package procurement

import (
	"context"
	"fmt"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// AcknowledgeReceivingDiscrepancyUseCase marks a single ReceivingDiscrepancy
// reviewed -- a human looked at it and decided no action is needed (e.g. a
// quantity_mismatch that's fine because the supplier always ships a bit
// over). Purely a review-tracking flag: it never touches stock, and
// CompleteReceivingUseCase doesn't require it (see ReceivingDiscrepancy's
// doc comment).
type AcknowledgeReceivingDiscrepancyUseCase struct {
	db   *gorm.DB
	repo procurementDomain.ReceivingDiscrepancyRepository
}

func NewAcknowledgeReceivingDiscrepancyUseCase(db *gorm.DB, repo procurementDomain.ReceivingDiscrepancyRepository) *AcknowledgeReceivingDiscrepancyUseCase {
	return &AcknowledgeReceivingDiscrepancyUseCase{db: db, repo: repo}
}

func (uc *AcknowledgeReceivingDiscrepancyUseCase) Execute(ctx context.Context, discrepancyID string) (*procurementDomain.ReceivingDiscrepancy, error) {
	var d *procurementDomain.ReceivingDiscrepancy
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		d, err = uc.repo.FindByID(ctx, discrepancyID)
		if err != nil {
			return err
		}
		if d.Status == procurementDomain.DiscrepancyStatusAcknowledged {
			return procurementDomain.ErrDiscrepancyAlreadyAcknowledged
		}
		now := time.Now()
		d.Status = procurementDomain.DiscrepancyStatusAcknowledged
		d.AcknowledgedAt = &now
		return uc.repo.Update(ctx, d)
	})
	if err != nil {
		return nil, fmt.Errorf("acknowledgeReceivingDiscrepancy: %w", err)
	}
	return d, nil
}
