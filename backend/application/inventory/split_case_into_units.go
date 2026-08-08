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
// The resulting unit stock is deliberately not batch-tracked (UnitInMovement
// always has BatchID nil, regardless of whether the case stock being split
// was itself lotted) -- carrying the case's lot/expiry over to the unit side
// would need each case to track its own sub-lot of units, which is a bigger
// feature on its own; for now the unit side is general stock.
type SplitCaseIntoUnitsUseCase struct {
	db           *gorm.DB
	skuRepo      pmsDomain.SKURepository
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

func NewSplitCaseIntoUnitsUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	movementRepo invDomain.StockMovementRepository,
	outboxRepo outbox.Repository,
) *SplitCaseIntoUnitsUseCase {
	return &SplitCaseIntoUnitsUseCase{db: db, skuRepo: skuRepo, movementRepo: movementRepo, outboxRepo: outboxRepo}
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
			TenantID: tenantID, SKUID: *caseSKU.CaseOfSKUID, WarehouseID: in.WarehouseID, BatchID: nil,
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
