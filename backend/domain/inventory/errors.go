package inventory

import "errors"

var (
	ErrWarehouseNotFound   = errors.New("warehouse not found")
	ErrBatchNotFound       = errors.New("batch not found")
	ErrInvalidQuantity     = errors.New("invalid quantity for movement type")
	ErrInvalidMovementType = errors.New("invalid movement type for this operation")
	ErrBatchHasMovements   = errors.New("batch has stock movements and cannot be deleted")
	// ErrInsufficientStock guards every movement that removes stock (sale,
	// transfer_out, damaged, lost, a negative adjustment) against the
	// specific batch/general ledger scope (sku_id + warehouse_id + batch_id,
	// batch_id nil = general) not actually having that much — the ledger is
	// append-only, so this is checked before the row is written, not after.
	ErrInsufficientStock = errors.New("insufficient stock for this batch/ledger")
	// ErrSKUIsNotACase guards SplitCaseIntoUnitsUseCase: the given SKU must
	// have pms.SKU.CaseOfSKUID/UnitsPerCase set (see that field's doc
	// comment) -- there's nothing to split otherwise.
	ErrSKUIsNotACase = errors.New("sku is not configured as a case of another sku")
)
