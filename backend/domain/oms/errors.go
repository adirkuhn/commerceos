package oms

import "errors"

var (
	ErrSalesOrderNotFound = errors.New("sales order not found")
	// ErrEmptyLineItems guards ReceivePOSSalesOrderUseCase -- a sale with
	// no line items has nothing to decrement stock for and is almost
	// certainly a caller bug, not a legitimate empty sale.
	ErrEmptyLineItems = errors.New("sales order must have at least one line item")
	// ErrAdhocLineItemRequiresName guards ReceivePOSSalesOrderUseCase's
	// adhoc-item path (a line with no SKUID) -- with no SKU to look a name
	// up from, Name is the only identifying data a line item like that
	// carries at all, so an empty one is rejected rather than silently
	// stored.
	ErrAdhocLineItemRequiresName = errors.New("adhoc line item requires a name")
	// ErrDuplicateExternalOrder guards SalesOrder's (TenantID, Source,
	// ExternalOrderID) unique constraint -- returned instead of a raw DB
	// error so a caller (ImportWooOrdersUseCase.createOne) can tell "lost
	// a create race against another concurrent import of the same order"
	// apart from a genuine failure, and fall back to treating it as an
	// update of the now-existing order instead of a hard error.
	ErrDuplicateExternalOrder = errors.New("sales order with this external id already exists")
	// ErrSalesOrderLineItemNotFound guards LinkLineItemSKUUseCase.
	ErrSalesOrderLineItemNotFound = errors.New("sales order line item not found")
	// ErrLineItemNotOnOrder guards LinkLineItemSKUUseCase/
	// RetryStockDecrementUseCase against a line item ID that exists but
	// belongs to a different sales order than the one named in the
	// request -- never trust the path's two IDs to agree without checking.
	ErrLineItemNotOnOrder = errors.New("line item does not belong to this sales order")
	// ErrOrderNotStockReducing guards RetryStockDecrementUseCase: retrying
	// a decrement only makes sense for an order whose current status
	// actually owes one (SalesOrderStatus.ReducesStock).
	ErrOrderNotStockReducing = errors.New("sales order's current status does not reduce stock")

	// ErrPickingSessionNotFound guards every picking use case's session lookup.
	ErrPickingSessionNotFound = errors.New("picking session not found")
	// ErrPickingSessionNotActive guards actions (scan, pause, complete)
	// against a session that's already completed -- mirrors
	// procurement.ErrBookingNotActive.
	ErrPickingSessionNotActive = errors.New("picking session is not active")
	// ErrOrderAlreadyFulfilled guards StartOrResumePickingUseCase against
	// starting a session for an order that's already fulfilled.
	ErrOrderAlreadyFulfilled = errors.New("sales order is already fulfilled")
	// ErrPickScanCodeOrSKURequired mirrors procurement.ErrScanCodeOrSKURequired.
	ErrPickScanCodeOrSKURequired = errors.New("either a scanned code or a sku_id is required")
	// ErrInvalidPickScanActionType guards ScanPickItemUseCase against an
	// action type outside PickScanActionType's known set, or one missing a
	// field it requires (e.g. new_item with no price).
	ErrInvalidPickScanActionType = errors.New("invalid pick scan action type")
	// ErrPickLineItemRequired guards a pick/substitute scan, which must
	// reference an existing SalesOrderLineItem.
	ErrPickLineItemRequired = errors.New("a line item is required for this action")
	// ErrLineItemSKUMismatch guards a 'pick' scan whose resolved SKU
	// doesn't match any line item on the order.
	ErrLineItemSKUMismatch = errors.New("scanned item does not match any line item on this order")
	// ErrPickScanNotFound guards DeletePickScanUseCase against a scan ID
	// that doesn't belong to the named session.
	ErrPickScanNotFound = errors.New("pick scan not found")
	// ErrSubstituteViaBatchEndpointOnly guards ScanPickItemUseCase against
	// the Substitute action type -- substitution now always goes through
	// SubstituteLineItemUseCase (one or more substitute lines at once), not
	// the single-scan endpoint.
	ErrSubstituteViaBatchEndpointOnly = errors.New("substitute action must use the substitute-batch endpoint")
	// ErrSubstituteLinesRequired guards SubstituteLineItemUseCase against an
	// empty batch -- nothing to substitute with.
	ErrSubstituteLinesRequired = errors.New("at least one substitute line is required")
	// ErrInvalidSubstituteQuantity guards SubstituteLineItemUseCase against
	// a substitute line whose SubstituteQuantity or OriginalQtyCovered
	// isn't a positive magnitude.
	ErrInvalidSubstituteQuantity = errors.New("substitute quantity and original quantity covered must be positive")
	// ErrSubstitutionCoverageMismatch guards SubstituteLineItemUseCase --
	// the batch's substitute lines, summed by OriginalQtyCovered, must
	// cover the original line item's own Quantity exactly, no more, no
	// less.
	ErrSubstitutionCoverageMismatch = errors.New("substitute lines must cover exactly the original line item's quantity")

	// ErrInvalidLineItemQuantity guards AddSalesOrderLineItemUseCase.
	ErrInvalidLineItemQuantity = errors.New("quantity must be positive")
	// ErrInvalidPrice guards AddSalesOrderLineItemUseCase against a
	// negative price -- 0 is valid (a freebie).
	ErrInvalidPrice = errors.New("price must not be negative")
	// ErrPriceRequiredForNewItem guards ScanPickItemUseCase's new_item
	// action -- unlike freebie (always forced to 0), a new item has no
	// default price to fall back to.
	ErrPriceRequiredForNewItem = errors.New("price is required for a new item")

	// ErrBulkUpdateOrderIDsRequired guards BulkUpdateSalesOrderStatusUseCase
	// against an empty selection -- a bulk action bar with nothing checked
	// has nothing to apply to.
	ErrBulkUpdateOrderIDsRequired = errors.New("at least one order id is required")
	// ErrBulkUpdateNoFieldsGiven guards BulkUpdateSalesOrderStatusUseCase
	// against a call that sets neither Status nor Fulfilled -- same
	// "at least one field" contract as a partial-update PATCH, just applied
	// before the loop instead of per-order.
	ErrBulkUpdateNoFieldsGiven = errors.New("at least one of status or fulfilled is required")
)
