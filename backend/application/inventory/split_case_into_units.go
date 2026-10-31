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
//
// The units carry the case's cost: the case batch's landed cost when the
// split is from a costed lot, else the case SKU's current cost, spread
// over the units (pmsDomain.UnitCostFromCaseMinorUnits). A new unit batch
// keeps that as its landed cost, and once the split commits the units are
// handed to costUpdater as a receipt, blending into the unit SKU's
// moving-average cost. No known cost: the unit's cost is left alone.
type SplitCaseIntoUnitsUseCase struct {
	db           *gorm.DB
	skuRepo      pmsDomain.SKURepository
	batchRepo    splitBatchStore
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
	// costUpdater is optional (nil: no cost follows the units).
	costUpdater skuCostUpdaterForReceiveStock
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
	costUpdater skuCostUpdaterForReceiveStock,
) *SplitCaseIntoUnitsUseCase {
	return &SplitCaseIntoUnitsUseCase{db: db, skuRepo: skuRepo, batchRepo: batchRepo, movementRepo: movementRepo, outboxRepo: outboxRepo, costUpdater: costUpdater}
}

func (uc *SplitCaseIntoUnitsUseCase) Execute(ctx context.Context, in SplitCaseIntoUnitsInput) (*SplitCaseIntoUnitsResult, error) {
	if in.CasesToSplit <= 0 {
		return nil, fmt.Errorf("splitCaseIntoUnits: %w", invDomain.ErrInvalidQuantity)
	}

	var result SplitCaseIntoUnitsResult
	// unitCost is the cost per unit the split brings in; nil when unknown.
	var unitCost *int64
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

		var caseBatch *invDomain.Batch
		if in.BatchID != nil {
			if caseBatch, err = uc.caseBatch(ctx, in); err != nil {
				return err
			}
		}
		unitCost = splitUnitCost(*caseSKU, caseBatch)

		unitBatchID, err := uc.resolveUnitBatch(ctx, caseBatch, *caseSKU.CaseOfSKUID, unitCost)
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

	if unitCost != nil && uc.costUpdater != nil {
		occurredAt := in.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		// Its own transaction after the split committed (tenant.WithTenant
		// must never nest) -- same accepted gap as ReceiveStockUseCase: a
		// failure leaves the unit cost behind, the split itself stands.
		if err := uc.costUpdater.Execute(ctx, SKUCostReceiptInput{
			SKUID: result.UnitInMovement.SKUID, ReceivedQuantity: result.UnitInMovement.QuantityDelta,
			LandedUnitCostMinorUnits: *unitCost,
			ReferenceType:            "stock_movement", ReferenceID: result.UnitInMovement.ID,
			OccurredAt: occurredAt,
		}); err != nil {
			return &result, fmt.Errorf("splitCaseIntoUnits: cost update: %w", err)
		}
	}
	return &result, nil
}

// splitUnitCost is what one split-out unit costs: the case lot's landed
// cost when known, else the case SKU's current cost, per unit; nil when
// neither is known.
func splitUnitCost(caseSKU pmsDomain.SKU, caseBatch *invDomain.Batch) *int64 {
	caseCost := caseSKU.CostPriceMinorUnits
	if caseBatch != nil && caseBatch.LandedCostMinorUnits != nil {
		caseCost = caseBatch.LandedCostMinorUnits
	}
	if caseCost == nil {
		return nil
	}
	cost := pmsDomain.UnitCostFromCaseMinorUnits(*caseCost, *caseSKU.UnitsPerCase)
	return &cost
}

// caseBatch loads the batch a split takes cases from, checking it is the
// case SKU's and in the split's warehouse.
func (uc *SplitCaseIntoUnitsUseCase) caseBatch(ctx context.Context, in SplitCaseIntoUnitsInput) (*invDomain.Batch, error) {
	b, err := uc.batchRepo.FindByID(ctx, *in.BatchID)
	if err != nil {
		return nil, err
	}
	if b.SKUID != in.CaseSKUID {
		return nil, invDomain.ErrBatchDoesNotBelongToSKU
	}
	if b.WarehouseID != in.WarehouseID {
		return nil, invDomain.ErrBatchNotInWarehouse
	}
	return b, nil
}

// resolveUnitBatch returns the unit-side batch ID for a split: nil for a
// general-ledger split (caseBatch nil), otherwise the unit SKU's active
// batch in the same warehouse with the same lot number and expiry date --
// created (copying the case batch's lot, expiry and receipt date, at
// unitCost) when none exists yet, so repeated splits of one lot top up a
// single unit batch.
func (uc *SplitCaseIntoUnitsUseCase) resolveUnitBatch(ctx context.Context, caseBatch *invDomain.Batch, unitSKUID string, unitCost *int64) (*string, error) {
	if caseBatch == nil {
		return nil, nil
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
		LandedCostMinorUnits: unitCost,
		Status:               invDomain.BatchStatusActive, CreatedAt: now, UpdatedAt: now,
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
