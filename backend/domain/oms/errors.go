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
)
