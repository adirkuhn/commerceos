package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type SplitCaseIntoUnitsInput struct {
	CaseSKUID    string
	WarehouseID  string
	BatchID      *string // nil = general/un-lotted case stock
	CasesToSplit int64   // always positive
	Reason       string
	OccurredAt   time.Time
}

type SplitCaseIntoUnitsResult struct {
	CaseOutMovement invDomain.StockMovement
	UnitInMovement  invDomain.StockMovement
}

// SplitCaseIntoUnitsUseCase converts sealed-case stock into individual-unit
// stock: CaseSKUID must be configured as a case of another SKU (see
// pms.SKU.CaseOfSKUID/UnitsPerCase) -- N cases leave stock (split_out) and
// N*UnitsPerCase units of the case's own SKU arrive (split_in), atomically.
// The reverse (assembling loose units back into cases) isn't implemented --
// this is a one-way "break the seal" operation, matching the real-world
// action it represents.
//
// Splitting from a batch keeps the lot: the units land in a batch on the
// unit SKU with the same warehouse, lot number and expiry date (see
// resolveUnitBatch) -- so FEFO, expiry alerts and batch-tied promotions
// keep seeing them. Splitting from the general ledger lands the units in
// the unit SKU's general ledger.
type SplitCaseIntoUnitsUseCase struct {
	db           *gorm.DB
	skuRepo      pmsDomain.SKURepository
	batchRepo    splitBatchStore
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

// splitBatchStore is the slice of invDomain.BatchRepository a split needs
// (ISP): read the case batch, find or create its unit-side twin.
type splitBatchStore interface {
	FindByID(ctx context.Context, id string) (*invDomain.Batch, error)
	ListBySKU(ctx context.Context, skuID string) ([]invDomain.Batch, error)
	Create(ctx context.Context, b *invDomain.Batch) error
}

func NewSplitCaseIntoUnitsUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	batchRepo splitBatchStore,
	movementRepo invDomain.StockMovementRepository,
	outboxRepo outbox.Repository,
) *SplitCaseIntoUnitsUseCase {
	return &SplitCaseIntoUnitsUseCase{db: db, skuRepo: skuRepo, batchRepo: batchRepo, movementRepo: movementRepo, outboxRepo: outboxRepo}
}

func (uc *SplitCaseIntoUnitsUseCase) Execute(ctx context.Context, in SplitCaseIntoUnitsInput) (*SplitCaseIntoUnitsResult, error) {
	if in.CasesToSplit <= 0 {
		return nil, fmt.Errorf("splitCaseIntoUnits: %w", invDomain.ErrInvalidQuantity)
	}

	var result SplitCaseIntoUnitsResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		caseSKU, err := uc.skuRepo.FindByID(ctx, in.CaseSKUID)
		if err != nil {
			return err
		}
		if caseSKU.CaseOfSKUID == nil || caseSKU.UnitsPerCase == nil {
			return invDomain.ErrSKUIsNotACase
		}

		movements, err := uc.movementRepo.ListBySKU(ctx, in.CaseSKUID)
		if err != nil {
			return err
		}
		if err := requireSufficientStock(movements, in.WarehouseID, in.BatchID, -in.CasesToSplit); err != nil {
			return err
		}

		unitBatchID, err := uc.resolveUnitBatch(ctx, in, *caseSKU.CaseOfSKUID)
		if err != nil {
			return err
		}

		now := time.Now()
		tenantID := tenant.IDFromContext(ctx)

		caseOut := invDomain.StockMovement{
			TenantID: tenantID, SKUID: in.CaseSKUID, WarehouseID: in.WarehouseID, BatchID: in.BatchID,
			Type: invDomain.MovementTypeSplitOut, QuantityDelta: -in.CasesToSplit,
			Reason: in.Reason, OccurredAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.movementRepo.Create(ctx, &caseOut); err != nil {
			return err
		}
		result.CaseOutMovement = caseOut

		unitIn := invDomain.StockMovement{
			TenantID: tenantID, SKUID: *caseSKU.CaseOfSKUID, WarehouseID: in.WarehouseID, BatchID: unitBatchID,
			Type: invDomain.MovementTypeSplitIn, QuantityDelta: in.CasesToSplit * *caseSKU.UnitsPerCase,
			Reason: in.Reason, OccurredAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.movementRepo.Create(ctx, &unitIn); err != nil {
			return err
		}
		result.UnitInMovement = unitIn

		if err := enqueueStockChanged(ctx, uc.outboxRepo, in.CaseSKUID); err != nil {
			return err
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, *caseSKU.CaseOfSKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("splitCaseIntoUnits: %w", err)
	}
	return &result, nil
}

// resolveUnitBatch returns the unit-side batch ID for a split: nil for a
// general-ledger split, otherwise the unit SKU's active batch in the same
// warehouse with the same lot number and expiry date -- created (copying
// the case batch's lot, expiry and receipt date) when none exists yet, so
// repeated splits of one lot top up a single unit batch.
func (uc *SplitCaseIntoUnitsUseCase) resolveUnitBatch(ctx context.Context, in SplitCaseIntoUnitsInput, unitSKUID string) (*string, error) {
	if in.BatchID == nil {
		return nil, nil
	}
	caseBatch, err := uc.batchRepo.FindByID(ctx, *in.BatchID)
	if err != nil {
		return nil, err
	}
	if caseBatch.SKUID != in.CaseSKUID {
		return nil, invDomain.ErrBatchDoesNotBelongToSKU
	}
	if caseBatch.WarehouseID != in.WarehouseID {
		return nil, invDomain.ErrBatchNotInWarehouse
	}

	unitBatches, err := uc.batchRepo.ListBySKU(ctx, unitSKUID)
	if err != nil {
		return nil, err
	}
	for _, b := range unitBatches {
		if b.Status == invDomain.BatchStatusActive && b.WarehouseID == caseBatch.WarehouseID &&
			b.LotNumber == caseBatch.LotNumber && sameExpiry(b.ExpiryDate, caseBatch.ExpiryDate) {
			return &b.ID, nil
		}
	}

	now := time.Now()
	unitBatch := invDomain.Batch{
		TenantID: tenant.IDFromContext(ctx), SKUID: unitSKUID, WarehouseID: caseBatch.WarehouseID,
		LotNumber: caseBatch.LotNumber, ExpiryDate: caseBatch.ExpiryDate, ReceivedAt: caseBatch.ReceivedAt,
		Status: invDomain.BatchStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.batchRepo.Create(ctx, &unitBatch); err != nil {
		return nil, err
	}
	return &unitBatch.ID, nil
}

func sameExpiry(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
