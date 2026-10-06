package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// FlagDamagedLineItemUseCase records a human's physical-inspection finding
// that some received quantity of a PO line item arrived damaged -- unlike
// the other discrepancy types, damage can never be inferred from the
// invoice text, so this creates a ReceivingDiscrepancy row directly instead
// of coming out of diffInvoiceAgainstPurchaseOrder.
type FlagDamagedLineItemUseCase struct {
	db              *gorm.DB
	recRepo         procurementDomain.ReceivingRecordRepository
	lineItemRepo    procurementDomain.PurchaseOrderLineItemRepository
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository
	outbox          eventrecord.Enqueuer
}

func NewFlagDamagedLineItemUseCase(
	db *gorm.DB,
	recRepo procurementDomain.ReceivingRecordRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository,
	outbox eventrecord.Enqueuer,
) *FlagDamagedLineItemUseCase {
	return &FlagDamagedLineItemUseCase{db: db, recRepo: recRepo, lineItemRepo: lineItemRepo, discrepancyRepo: discrepancyRepo, outbox: outbox}
}

func (uc *FlagDamagedLineItemUseCase) Execute(ctx context.Context, receivingRecordID, lineItemID string, damagedQty int64, note string) (*procurementDomain.ReceivingDiscrepancy, error) {
	if damagedQty <= 0 {
		return nil, procurementDomain.ErrInvalidDamagedQuantity
	}

	var d *procurementDomain.ReceivingDiscrepancy
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		rec, err := uc.recRepo.FindByID(ctx, receivingRecordID)
		if err != nil {
			return err
		}
		li, err := uc.lineItemRepo.FindByID(ctx, lineItemID)
		if err != nil {
			return err
		}
		if li.PurchaseOrderID != rec.PurchaseOrderID {
			return procurementDomain.ErrDiscrepancyMismatch
		}

		id := lineItemID
		d = &procurementDomain.ReceivingDiscrepancy{
			TenantID: tenant.IDFromContext(ctx), ReceivingRecordID: receivingRecordID, LineItemID: &id,
			Type: procurementDomain.DiscrepancyTypeDamagedItem, Status: procurementDomain.DiscrepancyStatusOpen,
			ExpectedQty: 0, ReceivedQty: damagedQty, Details: note, CreatedAt: time.Now(),
		}
		if err := uc.discrepancyRepo.Create(ctx, d); err != nil {
			return err
		}
		return recordDiscrepancyDetected(ctx, uc.outbox, *d, DiscrepancySourceManual)
	})
	if err != nil {
		return nil, fmt.Errorf("flagDamagedLineItem: %w", err)
	}
	return d, nil
}
