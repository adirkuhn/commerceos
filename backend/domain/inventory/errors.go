package inventory

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrWarehouseNotFound = errs.New("WAREHOUSE_NOT_FOUND", "warehouse not found")
	// ErrDuplicateWarehouseName guards Warehouse.Name: unlike suppliers
	// (OCR-extracted, name variance expected, matched by human review),
	// warehouses are staff-created directly, so a duplicate name within a
	// tenant is just picker/dropdown confusion (warehouses_tenant_id_name_key).
	ErrDuplicateWarehouseName = errs.New("WAREHOUSE_NAME_TAKEN", "warehouse name already in use")
	ErrBatchNotFound          = errs.New("BATCH_NOT_FOUND", "batch not found")
	ErrInvalidQuantity        = errs.New("MOVEMENT_INVALID_QUANTITY", "invalid quantity for movement type")
	ErrInvalidMovementType    = errs.New("MOVEMENT_INVALID_TYPE", "invalid movement type for this operation")
	ErrBatchHasMovements      = errs.New("BATCH_HAS_MOVEMENTS", "batch has stock movements and cannot be deleted")
	// ErrInsufficientStock guards every movement that removes stock (sale,
	// transfer_out, damaged, lost, a negative adjustment) against the
	// specific batch/general ledger scope (sku_id + warehouse_id + batch_id,
	// batch_id nil = general) not actually having that much — the ledger is
	// append-only, so this is checked before the row is written, not after.
	ErrInsufficientStock = errs.New("BATCH_INSUFFICIENT_STOCK", "insufficient stock for this batch/ledger")
	// ErrSKUIsNotACase guards SplitCaseIntoUnitsUseCase: the given SKU must
	// have pms.SKU.CaseOfSKUID/UnitsPerCase set (see that field's doc
	// comment) -- there's nothing to split otherwise.
	ErrSKUIsNotACase = errs.New("SKU_IS_NOT_A_CASE", "sku is not configured as a case of another sku")
	// ErrSKUIsNotPortioned guards PortionFromBulkUseCase: the given SKU
	// must have pms.SKU.PortionedFromSKUID set -- there's no bulk pool to
	// consume from otherwise.
	ErrSKUIsNotPortioned = errs.New("SKU_IS_NOT_PORTIONED", "sku is not configured as portioned from a bulk sku")
	// ErrUnsupportedPhotoType guards DiscardBatchUseCase's optional photo
	// upload -- same allow-list reasoning as pms.ErrUnsupportedMediaType.
	ErrUnsupportedPhotoType = errs.New("PHOTO_TYPE_UNSUPPORTED", "unsupported photo content type")
	// ErrBatchDoesNotBelongToSKU guards any caller that looks up a batch by
	// ID expecting it to belong to a specific SKU (e.g.
	// pms.PackWeightPortionsUseCase resolving the bulk lot to draw
	// from) -- a batch ID for a different SKU is a caller/client bug, not
	// a "not found".
	ErrBatchDoesNotBelongToSKU = errs.New("BATCH_DOES_NOT_BELONG_TO_SKU", "batch does not belong to this sku")
	// ErrBatchNotInWarehouse guards a caller naming both a warehouse and a
	// batch (e.g. SplitCaseIntoUnitsUseCase) when the batch lives in a
	// different warehouse than the one given.
	ErrBatchNotInWarehouse = errs.New("BATCH_NOT_IN_WAREHOUSE", "batch is not in this warehouse")
)
