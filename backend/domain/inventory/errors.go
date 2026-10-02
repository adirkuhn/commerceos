package inventory

import "errors"

var (
	ErrWarehouseNotFound = errors.New("warehouse not found")
	// ErrDuplicateWarehouseName guards Warehouse.Name: unlike suppliers
	// (OCR-extracted, name variance expected, matched by human review),
	// warehouses are staff-created directly, so a duplicate name within a
	// tenant is just picker/dropdown confusion (warehouses_tenant_id_name_key).
	ErrDuplicateWarehouseName = errors.New("warehouse name already in use")
	ErrBatchNotFound          = errors.New("batch not found")
	ErrInvalidQuantity        = errors.New("invalid quantity for movement type")
	ErrInvalidMovementType    = errors.New("invalid movement type for this operation")
	ErrBatchHasMovements      = errors.New("batch has stock movements and cannot be deleted")
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
	// ErrSKUIsNotPortioned guards PortionFromBulkUseCase: the given SKU
	// must have pms.SKU.PortionedFromSKUID set -- there's no bulk pool to
	// consume from otherwise.
	ErrSKUIsNotPortioned = errors.New("sku is not configured as portioned from a bulk sku")
	// ErrUnsupportedPhotoType guards DiscardBatchUseCase's optional photo
	// upload -- same allow-list reasoning as pms.ErrUnsupportedMediaType.
	ErrUnsupportedPhotoType = errors.New("unsupported photo content type")
	// ErrBatchDoesNotBelongToSKU guards any caller that looks up a batch by
	// ID expecting it to belong to a specific SKU (e.g.
	// pms.PackWeightPortionsUseCase resolving the bulk lot to draw
	// from) -- a batch ID for a different SKU is a caller/client bug, not
	// a "not found".
	ErrBatchDoesNotBelongToSKU = errors.New("batch does not belong to this sku")
	// ErrBatchNotInWarehouse guards a caller naming both a warehouse and a
	// batch (e.g. SplitCaseIntoUnitsUseCase) when the batch lives in a
	// different warehouse than the one given.
	ErrBatchNotInWarehouse = errors.New("batch is not in this warehouse")
)
