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
	// ErrPickingSessionLocked guards StartOrResumePickingUseCase/
	// TouchPickingSessionUseCase against a session already locked by a
	// different user within PickingLockTTL -- two people must not work the
	// same order's picking session at once.
	ErrPickingSessionLocked = errors.New("picking session is locked by another user")
	// ErrPickingSessionAlreadyCompleted guards ResetPickingSessionUseCase --
	// completion is a one-way door (CompletePickingUseCase's replay already
	// made staged rows real), so there's nothing left to reset.
	ErrPickingSessionAlreadyCompleted = errors.New("picking session is already completed")
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
	// cover the line item's own REMAINING quantity exactly (its Quantity
	// minus whatever Pick/Substitute/Refund scans already staged against
	// it this session), no more, no less. Not the full original Quantity
	// -- a line already partially picked can still have the rest of it
	// substituted.
	ErrSubstitutionCoverageMismatch = errors.New("substitute lines must cover exactly the line item's remaining (unpicked) quantity")
	// ErrInvalidRefundQuantity guards RefundLineItemUseCase against a
	// quantity that isn't a positive magnitude, or that exceeds the line
	// item's own Quantity.
	ErrInvalidRefundQuantity = errors.New("refund quantity must be positive and not exceed the line item's quantity")
	// ErrRefundRequiresLinkedSKU guards RefundLineItemUseCase against a
	// line item with no SKUID (an unmatched Woo import line) -- there's no
	// SKU to attribute the refund's PickScan row to.
	ErrRefundRequiresLinkedSKU = errors.New("line item must be linked to a sku before it can be refunded")
	// ErrPackingSessionNotFound guards every packing use case's session lookup.
	ErrPackingSessionNotFound = errors.New("packing session not found")
	// ErrPackingSessionNotActive guards actions (scan, complete) against a
	// packing session that's already completed.
	ErrPackingSessionNotActive = errors.New("packing session is not active")
	// ErrPackingSessionLocked mirrors ErrPickingSessionLocked, for packing.
	ErrPackingSessionLocked = errors.New("packing session is locked by another user")
	// ErrPackingSessionAlreadyCompleted guards ResetPackingSessionUseCase --
	// completion is a one-way door here too, same reasoning as
	// ErrPickingSessionAlreadyCompleted.
	ErrPackingSessionAlreadyCompleted = errors.New("packing session is already completed")
	// ErrOrderNotPackable guards SalesOrder.CanStartPacking -- an order that
	// skipped picking may only go straight to packing while its status is
	// still pickable (see SalesOrderStatus.IsPickable).
	ErrOrderNotPackable = errors.New("sales order status does not allow packing")
	// ErrOrderPickingInProgress guards SalesOrder.CanStartPacking against an
	// order with an open (in_progress/paused) picking session -- finish or
	// reset that session first.
	ErrOrderPickingInProgress = errors.New("sales order has a picking session in progress")
	// ErrPickingSessionHasPendingChanges guards StartOrResumePackingUseCase's
	// takeover of an open picking session that still holds unapplied
	// substitute/refund/add changes -- abandoning it would silently drop
	// them, so the picking session must be completed or reset first.
	ErrPickingSessionHasPendingChanges = errors.New("picking session has unapplied changes; complete or reset it first")
	// ErrOrderPackingInProgress guards SalesOrder.CanStartPicking against an
	// order already being packed.
	ErrOrderPackingInProgress = errors.New("sales order has a packing session in progress")
	// ErrPickListOrderIDsRequired guards BuildPickListUseCase against an
	// empty selection.
	ErrPickListOrderIDsRequired = errors.New("at least one sales order id is required")
	// ErrPickListTooManyOrders guards BuildPickListUseCase against a
	// selection larger than MaxPickListOrders.
	ErrPickListTooManyOrders = errors.New("too many sales orders for one pick list")
	// ErrOrderAlreadyPacked guards StartOrResumePackingUseCase against
	// starting a new packing session for an order that's already packed.
	ErrOrderAlreadyPacked = errors.New("sales order is already packed")
	// ErrPackScanNotFound guards DeletePackScanUseCase against a scan ID
	// that doesn't belong to the named session.
	ErrPackScanNotFound = errors.New("pack scan not found")
	// ErrPackScanCodeOrSKURequired mirrors ErrPickScanCodeOrSKURequired.
	ErrPackScanCodeOrSKURequired = errors.New("either a scanned code or a sku_id is required")
	// ErrPackScanSKUMismatch guards ScanPackItemUseCase against a scanned
	// SKU that isn't linked to any of this order's line items.
	ErrPackScanSKUMismatch = errors.New("scanned item does not match any line item on this order")
	// ErrPackQuantityExceedsOrdered guards ScanPackItemUseCase against
	// packing more of a line item than the order actually contains.
	ErrPackQuantityExceedsOrdered = errors.New("packed quantity would exceed the line item's ordered quantity")

	// ErrPermissionDenied guards a use case's fine-grained RBAC check (see
	// iam.GroupPermissionRepository.HasPermission) against a customer_user
	// caller who lacks the specific rule required -- e.g.
	// RefundLineItemUseCase's "oms.line_item.refund". Never raised for
	// customer_owner/customer_admin/super_admin, which bypass fine-grained
	// checks entirely (full tenant access by role).
	ErrPermissionDenied = errors.New("caller does not have permission to perform this action")

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
