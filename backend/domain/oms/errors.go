package oms

import "errors"

var (
	ErrSalesOrderNotFound = errors.New("sales order not found")
	// ErrEmptyLineItems guards ReceivePOSSalesOrderUseCase -- a sale with
	// no line items has nothing to decrement stock for and is almost
	// certainly a caller bug, not a legitimate empty sale.
	ErrEmptyLineItems = errors.New("sales order must have at least one line item")
)
