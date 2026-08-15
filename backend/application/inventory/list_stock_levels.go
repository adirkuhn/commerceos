package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultStockLevelsPerPage = 50
	maxStockLevelsPerPage     = 500
)

// ListStockLevelsInput's ChangedSince is the incremental-sync filter an
// external integration (e.g. a POS desktop) uses to pull only what moved
// since its last sync -- "changed" means "has a StockMovement after this
// time", not a timestamped field on the SKU itself (stock has no such
// field; it's derived from the movement ledger).
type ListStockLevelsInput struct {
	ChangedSince *time.Time
	Page         int
	PerPage      int
}

type ListStockLevelsResult struct {
	Levels []StockLevel
	Total  int64
}

// ListStockLevelsUseCase is the bulk counterpart to GetSKUStockLevelUseCase
// -- pages through every SKU with stock history (or, filtered, every SKU
// that changed since a given time), computing each one's level with the
// same shared computeStockLevel rule. Fetches movements/batches for the
// whole page's SKUs in two bulk queries rather than one pair of queries per
// SKU.
type ListStockLevelsUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
	settingsRepo settingsGetter
}

func NewListStockLevelsUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository, settingsRepo settingsGetter) *ListStockLevelsUseCase {
	return &ListStockLevelsUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo, settingsRepo: settingsRepo}
}

func (uc *ListStockLevelsUseCase) Execute(ctx context.Context, in ListStockLevelsInput) (*ListStockLevelsResult, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	perPage := in.PerPage
	if perPage <= 0 {
		perPage = defaultStockLevelsPerPage
	}
	if perPage > maxStockLevelsPerPage {
		perPage = maxStockLevelsPerPage
	}

	result := &ListStockLevelsResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		skuIDs, total, err := uc.movementRepo.ListDistinctSKUIDs(ctx, in.ChangedSince, perPage, (page-1)*perPage)
		if err != nil {
			return err
		}
		result.Total = total
		if len(skuIDs) == 0 {
			return nil
		}

		movements, err := uc.movementRepo.ListBySKUs(ctx, skuIDs)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKUs(ctx, skuIDs)
		if err != nil {
			return err
		}
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}

		movementsBySKU := make(map[string][]invDomain.StockMovement, len(skuIDs))
		for _, m := range movements {
			movementsBySKU[m.SKUID] = append(movementsBySKU[m.SKUID], m)
		}
		batchesBySKU := make(map[string][]invDomain.Batch, len(skuIDs))
		for _, b := range batches {
			batchesBySKU[b.SKUID] = append(batchesBySKU[b.SKUID], b)
		}

		result.Levels = make([]StockLevel, len(skuIDs))
		for i, skuID := range skuIDs {
			result.Levels[i] = computeStockLevel(skuID, movementsBySKU[skuID], batchesBySKU[skuID], settings.InventorySyncRulesEnabled)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listStockLevels: %w", err)
	}
	return result, nil
}
