package inventory

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// AllocateSaleStockInput removes stock for a stock-out that has no specific
// batch reference of its own -- a Woo/POS/manual sales order line item,
// unlike receiving or a manual adjustment, doesn't know which physical
// batch it came from. See AllocateSaleStockUseCase.Execute for the picking
// order this spreads Quantity across.
type AllocateSaleStockInput struct {
	SKUID       string
	WarehouseID string
	// Quantity is positive -- the amount to remove. Unlike
	// RecordStockMovementInput.QuantityDelta, which is signed, every
	// movement this writes is derived as -Quantity (split across
	// whichever scopes were picked).
	Quantity   int64
	Type       invDomain.MovementType
	Reason     string
	OccurredAt time.Time
	// SuppressEvent skips enqueueing inventory.stock.changed -- see
	// RecordStockMovementInput.SuppressEvent.
	SuppressEvent bool
	// ReconcileShortfall mirrors RecordStockMovementInput.ReconcileShortfall,
	// applied to whatever remains after every real batch and the general
	// ledger are exhausted: an Adjustment movement for that remainder is
	// written against the general ledger first (recording that untracked
	// stock existed), then the sale itself proceeds against it too, instead
	// of failing. For a source where the removal already happened in the
	// real world (a completed POS sale) and the ledger just never saw a
	// matching receipt.
	ReconcileShortfall bool
}

type AllocateSaleStockUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
	outboxRepo   outbox.Repository
}

func NewAllocateSaleStockUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository, outboxRepo outbox.Repository) *AllocateSaleStockUseCase {
	return &AllocateSaleStockUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo, outboxRepo: outboxRepo}
}

type saleAllocationPlan struct {
	batchID *string
	qty     int64
}

// CheckAvailability reports how much of quantity would be short if Execute
// were called right now, without writing anything -- lets a caller (e.g.
// CompletePickingUseCase) detect a genuine shortfall up front and get a
// staff decision before committing to either resolution, instead of
// Execute's own hard failure (ErrInsufficientStock, nothing written) being
// the first anyone hears of it.
func (uc *AllocateSaleStockUseCase) CheckAvailability(ctx context.Context, skuID, warehouseID string, quantity int64) (short int64, err error) {
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		_, remaining := planAllocation(movements, batches, warehouseID, quantity)
		short = remaining
		return nil
	})
	return short, err
}

// Execute picks stock in this order, splitting across as many scopes as
// needed to cover Quantity: (1) active batches with an ExpiryDate,
// soonest-expiring first (FEFO -- sell what expires soonest); (2) active
// batches with no ExpiryDate, oldest-received first (FIFO); (3) the
// general/un-lotted ledger, last resort. Fails with
// invDomain.ErrInsufficientStock, writing nothing at all, only if the TRUE
// total across every one of those scopes is still short of Quantity --
// unlike RecordStockMovementUseCase's single-scope check (which only ever
// looks at one BatchID/the general ledger), this is what fixes stock
// appearing "insufficient" when it's really just sitting in a batch a sale
// never references.
func (uc *AllocateSaleStockUseCase) Execute(ctx context.Context, in AllocateSaleStockInput) ([]invDomain.StockMovement, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("allocateSaleStock: %w", invDomain.ErrInvalidQuantity)
	}
	if err := validateSign(in.Type, -in.Quantity); err != nil {
		return nil, fmt.Errorf("allocateSaleStock: %w", err)
	}

	var created []invDomain.StockMovement
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, in.SKUID)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKU(ctx, in.SKUID)
		if err != nil {
			return err
		}

		plans, remaining := planAllocation(movements, batches, in.WarehouseID, in.Quantity)
		now := time.Now()
		if remaining > 0 {
			if !in.ReconcileShortfall {
				log.Printf("[DEBUG] Insufficient stock across every batch/ledger for SKU %s in warehouse %s: needed %d, %d short even after checking every batch",
					in.SKUID, in.WarehouseID, in.Quantity, remaining)
				return invDomain.ErrInsufficientStock
			}
			adjustment := invDomain.StockMovement{
				TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID, WarehouseID: in.WarehouseID, BatchID: nil,
				Type: invDomain.MovementTypeAdjustment, QuantityDelta: remaining,
				Reason:     fmt.Sprintf("reconciliation: untracked stock found for %s", in.Reason),
				OccurredAt: in.OccurredAt, CreatedAt: now,
			}
			if err := uc.movementRepo.Create(ctx, &adjustment); err != nil {
				return err
			}
			plans = append(plans, saleAllocationPlan{batchID: nil, qty: remaining})
		}
		for _, p := range plans {
			m := invDomain.StockMovement{
				TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID, WarehouseID: in.WarehouseID,
				BatchID: p.batchID, Type: in.Type, QuantityDelta: -p.qty,
				Reason: in.Reason, OccurredAt: in.OccurredAt, CreatedAt: now,
			}
			if err := uc.movementRepo.Create(ctx, &m); err != nil {
				return err
			}
			created = append(created, m)
		}
		if in.SuppressEvent {
			return nil
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("allocateSaleStock: %w", err)
	}
	return created, nil
}

// planAllocation computes the allocation without writing anything --
// separated from Execute so insufficiency is known before a single
// StockMovement is created (matching RecordStockMovementUseCase's
// check-then-write shape). Returns the plan and however much of quantity
// is still unmet (0 means fully covered).
func planAllocation(movements []invDomain.StockMovement, batches []invDomain.Batch, warehouseID string, quantity int64) ([]saleAllocationPlan, int64) {
	var plans []saleAllocationPlan
	remaining := quantity
	for _, batchID := range allocationOrder(batches, warehouseID) {
		if remaining <= 0 {
			break
		}
		available := currentStock(movements, warehouseID, batchID)
		if available <= 0 {
			continue
		}
		take := min(available, remaining)
		plans = append(plans, saleAllocationPlan{batchID: batchID, qty: take})
		remaining -= take
	}
	return plans, remaining
}

// allocationOrder returns every candidate scope -- real batches in
// FEFO-then-FIFO order, then the general ledger (nil) last. Expired-status
// batches are excluded: can't sell what's already flagged expired.
func allocationOrder(batches []invDomain.Batch, warehouseID string) []*string {
	var withExpiry, withoutExpiry []invDomain.Batch
	for _, b := range batches {
		if b.WarehouseID != warehouseID || b.Status == invDomain.BatchStatusExpired {
			continue
		}
		if b.ExpiryDate != nil {
			withExpiry = append(withExpiry, b)
		} else {
			withoutExpiry = append(withoutExpiry, b)
		}
	}
	sort.Slice(withExpiry, func(i, j int) bool {
		return withExpiry[i].ExpiryDate.Before(*withExpiry[j].ExpiryDate)
	})
	sort.Slice(withoutExpiry, func(i, j int) bool {
		return withoutExpiry[i].ReceivedAt.Before(withoutExpiry[j].ReceivedAt)
	})

	out := make([]*string, 0, len(batches)+1)
	for _, b := range withExpiry {
		id := b.ID
		out = append(out, &id)
	}
	for _, b := range withoutExpiry {
		id := b.ID
		out = append(out, &id)
	}
	return append(out, nil)
}
