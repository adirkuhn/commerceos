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
)
