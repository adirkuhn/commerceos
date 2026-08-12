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

type PortionFromBulkInput struct {
	TargetSKUID string
	WarehouseID string
	BatchID     *string // nil = general/un-lotted bulk stock
	// WeightGrams is the actual amount consumed for this one physical
	// item (e.g. a scale reading) -- nil defaults to the target SKU's own
	// nominal WeightGrams.
	WeightGrams *int64
	Reason      string
	OccurredAt  time.Time
}

type PortionFromBulkResult struct {
	BulkOutMovement invDomain.StockMovement
	UnitInMovement  invDomain.StockMovement
}

// PortionFromBulkUseCase peels one sellable unit off a bulk (UnitType=kilo)
// SKU's weight pool: TargetSKUID must be configured as portioned from a
// bulk SKU (see pms.SKU.PortionedFromSKUID) -- WeightGrams leaves the bulk
// SKU's stock and 1 unit of the target SKU arrives, atomically.
//
// Unlike SplitCaseIntoUnitsUseCase's fixed N-cases -> N*UnitsPerCase
// multiplier, this is called once per physical item and is meant to be
// repeated indefinitely as items are portioned off the same bulk lot over
// time -- there's no case count here, just a running weight pool that
// shrinks by whatever each individual item actually weighs.
//
// The resulting unit stock is deliberately not batch-tracked (UnitInMovement
// always has BatchID nil), same reasoning as SplitCaseIntoUnitsUseCase.
type PortionFromBulkUseCase struct {
	db           *gorm.DB
	skuRepo      pmsDomain.SKURepository
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

func NewPortionFromBulkUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	movementRepo invDomain.StockMovementRepository,
	outboxRepo outbox.Repository,
) *PortionFromBulkUseCase {
	return &PortionFromBulkUseCase{db: db, skuRepo: skuRepo, movementRepo: movementRepo, outboxRepo: outboxRepo}
}

func (uc *PortionFromBulkUseCase) Execute(ctx context.Context, in PortionFromBulkInput) (*PortionFromBulkResult, error) {
	var result PortionFromBulkResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		target, err := uc.skuRepo.FindByID(ctx, in.TargetSKUID)
		if err != nil {
			return err
		}
		if target.PortionedFromSKUID == nil {
			return invDomain.ErrSKUIsNotPortioned
		}

		weightGrams := in.WeightGrams
		if weightGrams == nil {
			weightGrams = target.WeightGrams
		}
		if weightGrams == nil || *weightGrams <= 0 {
			return invDomain.ErrInvalidQuantity
		}

		bulkSKUID := *target.PortionedFromSKUID
		movements, err := uc.movementRepo.ListBySKU(ctx, bulkSKUID)
		if err != nil {
			return err
		}
		if err := requireSufficientStock(movements, in.WarehouseID, in.BatchID, -*weightGrams); err != nil {
			return err
		}

		now := time.Now()
		tenantID := tenant.IDFromContext(ctx)

		bulkOut := invDomain.StockMovement{
			TenantID: tenantID, SKUID: bulkSKUID, WarehouseID: in.WarehouseID, BatchID: in.BatchID,
			Type: invDomain.MovementTypeSplitOut, QuantityDelta: -*weightGrams,
			Reason: in.Reason, OccurredAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.movementRepo.Create(ctx, &bulkOut); err != nil {
			return err
		}
		result.BulkOutMovement = bulkOut

		unitIn := invDomain.StockMovement{
			TenantID: tenantID, SKUID: in.TargetSKUID, WarehouseID: in.WarehouseID, BatchID: nil,
			Type: invDomain.MovementTypeSplitIn, QuantityDelta: 1,
			Reason: in.Reason, OccurredAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.movementRepo.Create(ctx, &unitIn); err != nil {
			return err
		}
		result.UnitInMovement = unitIn

		if err := enqueueStockChanged(ctx, uc.outboxRepo, bulkSKUID); err != nil {
			return err
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, in.TargetSKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("portionFromBulk: %w", err)
	}
	return &result, nil
}
