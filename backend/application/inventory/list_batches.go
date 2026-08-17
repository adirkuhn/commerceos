package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListBatchesUseCase struct {
	db            *gorm.DB
	batchRepo     invDomain.BatchRepository
	movementRepo  invDomain.StockMovementRepository
	warehouseRepo invDomain.WarehouseRepository
}

func NewListBatchesUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, movementRepo invDomain.StockMovementRepository, warehouseRepo invDomain.WarehouseRepository) *ListBatchesUseCase {
	return &ListBatchesUseCase{db: db, batchRepo: batchRepo, movementRepo: movementRepo, warehouseRepo: warehouseRepo}
}

// Execute returns one row per warehouse the SKU has any general (un-lotted)
// activity in -- IsGeneralLedger:true, Quantity summed from batch_id=nil
// movements, listed first -- followed by every real batch with its own
// Quantity summed from its movements. Both are computed from the ledger,
// never stored (see Batch's doc comment).
//
// Every tenant warehouse gets a general-ledger row even with zero activity
// (quantity 0), not just ones the SKU already has history in -- a brand-new
// SKU would otherwise return an empty list, leaving no row for the UI to
// act on and no way to ever enter its first stock.
func (uc *ListBatchesUseCase) Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error) {
	var result []invDomain.Batch
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		batches, err := uc.batchRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		movements, err := uc.movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		warehouses, err := uc.warehouseRepo.List(ctx)
		if err != nil {
			return err
		}

		qtyByBatch := make(map[string]int64, len(batches))
		generalByWarehouse := make(map[string]int64)
		var generalWarehouseOrder []string
		seenGeneralWarehouse := make(map[string]bool)
		addGeneralWarehouse := func(id string) {
			if !seenGeneralWarehouse[id] {
				seenGeneralWarehouse[id] = true
				generalWarehouseOrder = append(generalWarehouseOrder, id)
			}
		}
		// A batch's own warehouse needs a visible general-ledger row even
		// when that warehouse has no un-lotted movements yet (still 0, not
		// absent) -- otherwise a reviewer can't see there's nothing to
		// remove from general before trying to. Seeded from batches first
		// so a warehouse that actually holds this SKU's batches sorts
		// before one that only ever had un-lotted stock.
		for _, b := range batches {
			addGeneralWarehouse(b.WarehouseID)
		}
		for _, m := range movements {
			if m.BatchID != nil {
				qtyByBatch[*m.BatchID] += m.QuantityDelta
				continue
			}
			addGeneralWarehouse(m.WarehouseID)
			generalByWarehouse[m.WarehouseID] += m.QuantityDelta
		}
		for _, w := range warehouses {
			addGeneralWarehouse(w.ID)
		}

		result = make([]invDomain.Batch, 0, len(batches)+len(generalWarehouseOrder))
		for _, whID := range generalWarehouseOrder {
			result = append(result, invDomain.Batch{
				SKUID: skuID, WarehouseID: whID, IsGeneralLedger: true, Quantity: generalByWarehouse[whID],
			})
		}
		for _, b := range batches {
			b.Quantity = qtyByBatch[b.ID]
			result = append(result, b)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listBatches: %w", err)
	}
	return result, nil
}
