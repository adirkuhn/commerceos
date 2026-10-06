package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// LinkReceivingDiscrepanciesUseCase is the deliberate escape hatch for cases
// diffInvoiceAgainstPurchaseOrder couldn't confidently resolve on its own:
// an unexpected_item (invoice line matching no PO line) and a missing_item
// (PO line matching no invoice line) that a human recognizes as the same
// product. Rather than lower the matcher's fuzzy-match threshold and risk
// false positives, this lets a person confirm the pairing explicitly --
// both discrepancies are then acknowledged. It does not learn a
// SupplierProductAlias or otherwise change future matching; that's a
// separate, bigger decision than a one-off review acknowledgment.
type LinkReceivingDiscrepanciesUseCase struct {
	db     *gorm.DB
	repo   procurementDomain.ReceivingDiscrepancyRepository
	outbox eventrecord.Enqueuer
}

func NewLinkReceivingDiscrepanciesUseCase(db *gorm.DB, repo procurementDomain.ReceivingDiscrepancyRepository, outbox eventrecord.Enqueuer) *LinkReceivingDiscrepanciesUseCase {
	return &LinkReceivingDiscrepanciesUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *LinkReceivingDiscrepanciesUseCase) Execute(ctx context.Context, unexpectedDiscrepancyID, missingDiscrepancyID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		unexpected, err := uc.repo.FindByID(ctx, unexpectedDiscrepancyID)
		if err != nil {
			return err
		}
		missing, err := uc.repo.FindByID(ctx, missingDiscrepancyID)
		if err != nil {
			return err
		}
		if unexpected.ReceivingRecordID != missing.ReceivingRecordID {
			return procurementDomain.ErrDiscrepancyMismatch
		}
		if unexpected.Type != procurementDomain.DiscrepancyTypeUnexpectedItem || missing.Type != procurementDomain.DiscrepancyTypeMissingItem {
			return procurementDomain.ErrDiscrepancyMismatch
		}
		if unexpected.Status == procurementDomain.DiscrepancyStatusAcknowledged || missing.Status == procurementDomain.DiscrepancyStatusAcknowledged {
			return procurementDomain.ErrDiscrepancyAlreadyAcknowledged
		}

		now := time.Now()
		unexpected.Status = procurementDomain.DiscrepancyStatusAcknowledged
		unexpected.AcknowledgedAt = &now
		if err := uc.repo.Update(ctx, unexpected); err != nil {
			return err
		}
		missing.Status = procurementDomain.DiscrepancyStatusAcknowledged
		missing.AcknowledgedAt = &now
		if err := uc.repo.Update(ctx, missing); err != nil {
			return err
		}
		if err := uc.recordLinked(ctx, unexpected, missing.ID); err != nil {
			return err
		}
		return uc.recordLinked(ctx, missing, unexpected.ID)
	})
	if err != nil {
		return fmt.Errorf("linkReceivingDiscrepancies: %w", err)
	}
	return nil
}

func (uc *LinkReceivingDiscrepanciesUseCase) recordLinked(ctx context.Context, d *procurementDomain.ReceivingDiscrepancy, linkedID string) error {
	return eventrecord.Record(ctx, uc.outbox, events.ReceivingDiscrepancyLinked, d.ID, eventrecord.Data{
		"receiving_record_id": d.ReceivingRecordID, "type": string(d.Type), "linked_discrepancy_id": linkedID,
	})
}
