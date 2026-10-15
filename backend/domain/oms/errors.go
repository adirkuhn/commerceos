package oms

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrSalesOrderNotFound = errs.New("SALES_ORDER_NOT_FOUND", "sales order not found")
	// ErrEmptyLineItems guards ReceivePOSSalesOrderUseCase -- a sale with
	// no line items has nothing to decrement stock for and is almost
	// certainly a caller bug, not a legitimate empty sale.
	ErrEmptyLineItems = errs.New("SALES_ORDER_NO_LINE_ITEMS", "sales order must have at least one line item")
	// ErrAdhocLineItemRequiresName guards ReceivePOSSalesOrderUseCase's
	// adhoc-item path (a line with no SKUID) -- with no SKU to look a name
	// up from, Name is the only identifying data a line item like that
	// carries at all, so an empty one is rejected rather than silently
	// stored.
	ErrAdhocLineItemRequiresName = errs.New("LINE_ITEM_NAME_REQUIRED", "adhoc line item requires a name")
	// ErrWeighedLineItemQuantityMustBeOne guards ReceivePOSSalesOrderUseCase:
	// a weighed line is one package, so a weight with any other quantity is
	// ambiguous (per unit or total?) and is rejected rather than guessed.
	ErrWeighedLineItemQuantityMustBeOne = errs.New("WEIGHED_LINE_ITEM_QUANTITY_MUST_BE_ONE", "a line item with weight_grams must have quantity 1")
	// ErrDuplicateExternalOrder guards SalesOrder's (TenantID, Source,
	// ExternalOrderID) unique constraint -- returned instead of a raw DB
	// error so a caller (ImportWooOrdersUseCase.createOne) can tell "lost
	// a create race against another concurrent import of the same order"
	// apart from a genuine failure, and fall back to treating it as an
	// update of the now-existing order instead of a hard error.
	ErrDuplicateExternalOrder = errs.New("SALES_ORDER_DUPLICATE_EXTERNAL", "sales order with this external id already exists")
	// ErrSalesOrderLineItemNotFound guards LinkLineItemSKUUseCase.
	ErrSalesOrderLineItemNotFound = errs.New("SALES_ORDER_LINE_ITEM_NOT_FOUND", "sales order line item not found")
	// ErrLineItemNotOnOrder guards LinkLineItemSKUUseCase/
	// RetryStockDecrementUseCase against a line item ID that exists but
	// belongs to a different sales order than the one named in the
	// request -- never trust the path's two IDs to agree without checking.
	ErrLineItemNotOnOrder = errs.New("SALES_ORDER_LINE_ITEM_NOT_ON_ORDER", "line item does not belong to this sales order")
	// ErrOrderNotStockReducing guards RetryStockDecrementUseCase: retrying
	// a decrement only makes sense for an order whose current status
	// actually owes one (SalesOrderStatus.ReducesStock).
	ErrOrderNotStockReducing = errs.New("ORDER_NOT_STOCK_REDUCING", "sales order's current status does not reduce stock")

	// ErrPickingSessionNotFound guards every picking use case's session lookup.
	ErrPickingSessionNotFound = errs.New("PICKING_SESSION_NOT_FOUND", "picking session not found")
	// ErrPickingSessionNotActive guards actions (scan, pause, complete)
	// against a session that's already completed -- mirrors
	// procurement.ErrBookingNotActive.
	ErrPickingSessionNotActive = errs.New("PICKING_SESSION_NOT_ACTIVE", "picking session is not active")
	// ErrOrderAlreadyFulfilled guards StartOrResumePickingUseCase against
	// starting a session for an order that's already fulfilled.
	ErrOrderAlreadyFulfilled = errs.New("ORDER_ALREADY_FULFILLED", "sales order is already fulfilled")
	// ErrPickScanCodeOrSKURequired mirrors procurement.ErrScanCodeOrSKURequired.
	ErrPickScanCodeOrSKURequired = errs.New("PICK_SCAN_CODE_OR_SKU_REQUIRED", "either a scanned code or a sku_id is required")
	// ErrInvalidPickScanActionType guards ScanPickItemUseCase against an
	// action type outside PickScanActionType's known set, or one missing a
	// field it requires (e.g. new_item with no price).
	ErrInvalidPickScanActionType = errs.New("PICK_SCAN_INVALID_ACTION", "invalid pick scan action type")
	// ErrPickLineItemRequired guards a pick/substitute scan, which must
	// reference an existing SalesOrderLineItem.
	ErrPickLineItemRequired = errs.New("PICK_LINE_ITEM_REQUIRED", "a line item is required for this action")
	// ErrLineItemSKUMismatch guards a 'pick' scan whose resolved SKU
	// doesn't match any line item on the order.
	ErrLineItemSKUMismatch = errs.New("PICK_SCAN_SKU_MISMATCH", "scanned item does not match any line item on this order")
	// ErrPickScanNotFound guards DeletePickScanUseCase against a scan ID
	// that doesn't belong to the named session.
	ErrPickScanNotFound = errs.New("PICK_SCAN_NOT_FOUND", "pick scan not found")
	// ErrPickingSessionLocked guards StartOrResumePickingUseCase/
	// TouchPickingSessionUseCase against a session already locked by a
	// different user within PickingLockTTL -- two people must not work the
	// same order's picking session at once.
	ErrPickingSessionLocked = errs.New("PICKING_SESSION_LOCKED", "picking session is locked by another user")
	// ErrPickingSessionAlreadyCompleted guards ResetPickingSessionUseCase --
	// completion is a one-way door (CompletePickingUseCase's replay already
	// made staged rows real), so there's nothing left to reset.
	ErrPickingSessionAlreadyCompleted = errs.New("PICKING_SESSION_ALREADY_COMPLETED", "picking session is already completed")
	// ErrSubstituteLinesRequired guards SubstituteLineItemUseCase against an
	// empty batch -- nothing to substitute with.
	ErrSubstituteLinesRequired = errs.New("SUBSTITUTE_LINES_REQUIRED", "at least one substitute line is required")
	// ErrInvalidSubstituteQuantity guards SubstituteLineItemUseCase against
	// a substitute line whose SubstituteQuantity or OriginalQtyCovered
	// isn't a positive magnitude.
	ErrInvalidSubstituteQuantity = errs.New("SUBSTITUTE_QUANTITY_INVALID", "substitute quantity and original quantity covered must be positive")
	// ErrSubstitutionCoverageMismatch guards SubstituteLineItemUseCase --
	// the batch's substitute lines, summed by OriginalQtyCovered, must
	// cover the line item's own REMAINING quantity exactly (its Quantity
	// minus whatever Pick/Substitute/Refund scans already staged against
	// it this session), no more, no less. Not the full original Quantity
	// -- a line already partially picked can still have the rest of it
	// substituted.
	ErrSubstitutionCoverageMismatch = errs.New("SUBSTITUTION_COVERAGE_MISMATCH", "substitute lines must cover exactly the line item's remaining (unpicked) quantity")
	// ErrInvalidRefundQuantity guards RefundLineItemUseCase against a
	// quantity that isn't a positive magnitude, or that exceeds the line
	// item's own Quantity.
	ErrInvalidRefundQuantity = errs.New("REFUND_QUANTITY_INVALID", "refund quantity must be positive and not exceed the line item's quantity")
	// ErrRefundRequiresLinkedSKU guards RefundLineItemUseCase against a
	// line item with no SKUID (an unmatched Woo import line) -- there's no
	// SKU to attribute the refund's PickScan row to.
	ErrRefundRequiresLinkedSKU = errs.New("REFUND_REQUIRES_LINKED_SKU", "line item must be linked to a sku before it can be refunded")
	// ErrPackingSessionNotFound guards every packing use case's session lookup.
	ErrPackingSessionNotFound = errs.New("PACKING_SESSION_NOT_FOUND", "packing session not found")
	// ErrPackingSessionNotActive guards actions (scan, complete) against a
	// packing session that's already completed.
	ErrPackingSessionNotActive = errs.New("PACKING_SESSION_NOT_ACTIVE", "packing session is not active")
	// ErrPackingSessionLocked mirrors ErrPickingSessionLocked, for packing.
	ErrPackingSessionLocked = errs.New("PACKING_SESSION_LOCKED", "packing session is locked by another user")
	// ErrPackingSessionAlreadyCompleted guards ResetPackingSessionUseCase --
	// completion is a one-way door here too, same reasoning as
	// ErrPickingSessionAlreadyCompleted.
	ErrPackingSessionAlreadyCompleted = errs.New("PACKING_SESSION_ALREADY_COMPLETED", "packing session is already completed")
	// ErrOrderNotPackable guards SalesOrder.CanStartPacking -- only a
	// pending/processing/on-hold order (see SalesOrderStatus.IsPickable) is
	// packed; a completed/shipped/cancelled one has already left or never
	// will.
	ErrOrderNotPackable = errs.New("ORDER_NOT_PACKABLE", "sales order status does not allow packing")
	// ErrOrderNotPickable guards SalesOrder.CanMarkPicked -- only a
	// pending/processing/on-hold order can still be picked.
	ErrOrderNotPickable = errs.New("ORDER_NOT_PICKABLE", "sales order status does not allow picking")
	// ErrPOSOrderNotFulfillable guards SalesOrder.CanStartPicking/
	// CanStartPacking -- a POS sale is handed over at the counter, never
	// picked or packed.
	ErrPOSOrderNotFulfillable = errs.New("POS_ORDER_NOT_FULFILLABLE", "point-of-sale orders are not picked or packed")
	// ErrOrderPickingInProgress guards SalesOrder.CanStartPacking against an
	// order with an open (in_progress/paused) picking session -- finish or
	// reset that session first.
	ErrOrderPickingInProgress = errs.New("ORDER_PICKING_IN_PROGRESS", "sales order has a picking session in progress")
	// ErrPickingSessionHasPendingChanges guards StartOrResumePackingUseCase's
	// takeover of an open picking session that still holds unapplied
	// substitute/refund/add changes -- abandoning it would silently drop
	// them, so the picking session must be completed or reset first.
	ErrPickingSessionHasPendingChanges = errs.New("PICKING_SESSION_HAS_PENDING_CHANGES", "picking session has unapplied changes; complete or reset it first")
	// ErrOrderPackingInProgress guards SalesOrder.CanStartPicking against an
	// order already being packed.
	ErrOrderPackingInProgress = errs.New("ORDER_PACKING_IN_PROGRESS", "sales order has a packing session in progress")
	// ErrPickListOrderIDsRequired guards BuildPickListUseCase against an
	// empty selection.
	ErrPickListOrderIDsRequired = errs.New("PICK_LIST_ORDERS_REQUIRED", "at least one sales order id is required")
	// ErrPickListTooManyOrders guards BuildPickListUseCase against a
	// selection larger than MaxPickListOrders.
	ErrPickListTooManyOrders = errs.New("PICK_LIST_TOO_MANY_ORDERS", "too many sales orders for one pick list")
	// ErrOrderAlreadyPacked guards StartOrResumePackingUseCase against
	// starting a new packing session for an order that's already packed.
	ErrOrderAlreadyPacked = errs.New("ORDER_ALREADY_PACKED", "sales order is already packed")
	// ErrPackScanNotFound guards DeletePackScanUseCase against a scan ID
	// that doesn't belong to the named session.
	ErrPackScanNotFound = errs.New("PACK_SCAN_NOT_FOUND", "pack scan not found")
	// ErrPackScanCodeOrSKURequired mirrors ErrPickScanCodeOrSKURequired.
	ErrPackScanCodeOrSKURequired = errs.New("PACK_SCAN_CODE_OR_SKU_REQUIRED", "either a scanned code or a sku_id is required")
	// ErrOrderChangesHappenInPacking guards ScanPickItemUseCase: picking only
	// picks -- substitutes, extra items and refunds are recorded in packing.
	ErrOrderChangesHappenInPacking = errs.New("ORDER_CHANGES_HAPPEN_IN_PACKING", "substitutes, extra items and refunds are recorded in packing, not picking")
	// ErrInvalidPackScanActionType guards ScanPackItemUseCase: a scan is a
	// plain pack, a freebie or a new item -- substitutes and refunds have
	// their own endpoints.
	ErrInvalidPackScanActionType = errs.New("PACK_SCAN_INVALID_ACTION", "pack scan action must be pack, freebie or new_item")
	// ErrPackScanSKUMismatch guards ScanPackItemUseCase against a scanned
	// SKU that isn't linked to any of this order's line items.
	ErrPackScanSKUMismatch = errs.New("PACK_SCAN_SKU_MISMATCH", "scanned item does not match any line item on this order")
	// ErrPackQuantityExceedsOrdered guards ScanPackItemUseCase against
	// packing more of a line item than the order actually contains.
	ErrPackQuantityExceedsOrdered = errs.New("PACK_QUANTITY_EXCEEDS_ORDERED", "packed quantity would exceed the line item's ordered quantity")

	// ErrPermissionDenied guards a use case's fine-grained RBAC check (see
	// iam.GroupPermissionRepository.HasPermission) against a customer_user
	// caller who lacks the specific rule required -- e.g.
	// RefundLineItemUseCase's "oms.line_item.refund". Never raised for
	// customer_owner/customer_admin/super_admin, which bypass fine-grained
	// checks entirely (full tenant access by role).
	ErrPermissionDenied = errs.New("ORDER_ACTION_NOT_PERMITTED", "caller does not have permission to perform this action")

	// ErrInvalidLineItemQuantity guards AddSalesOrderLineItemUseCase.
	ErrInvalidLineItemQuantity = errs.New("LINE_ITEM_QUANTITY_INVALID", "quantity must be positive")
	// ErrInvalidPrice guards AddSalesOrderLineItemUseCase against a
	// negative price -- 0 is valid (a freebie).
	ErrInvalidPrice = errs.New("LINE_ITEM_PRICE_NEGATIVE", "price must not be negative")
	// ErrPriceRequiredForNewItem guards ScanPickItemUseCase's new_item
	// action -- unlike freebie (always forced to 0), a new item has no
	// default price to fall back to.
	ErrPriceRequiredForNewItem = errs.New("NEW_ITEM_PRICE_REQUIRED", "price is required for a new item")

	// ErrBulkUpdateOrderIDsRequired guards BulkUpdateSalesOrderStatusUseCase
	// against an empty selection -- a bulk action bar with nothing checked
	// has nothing to apply to.
	ErrBulkUpdateOrderIDsRequired = errs.New("BULK_UPDATE_ORDERS_REQUIRED", "at least one order id is required")
	// ErrBulkUpdateNoFieldsGiven guards BulkUpdateSalesOrderStatusUseCase
	// against a call that sets neither Status nor Fulfilled -- same
	// "at least one field" contract as a partial-update PATCH, just applied
	// before the loop instead of per-order.
	ErrBulkUpdateNoFieldsGiven = errs.New("BULK_UPDATE_NOTHING_TO_CHANGE", "at least one of status or fulfilled is required")
)
