package oms

import "github.com/adirkuhn/commerceos/domain/errs"

// ErrSalesOrderNotEditable: only an order made here in the admin, not yet
// fulfilled and still pending, processing or on hold, can be edited --
// a channel's order belongs to that channel, and a shipped one is history.
var ErrSalesOrderNotEditable = errs.New("SALES_ORDER_NOT_EDITABLE", "this sales order can no longer be edited")

// Editable reports whether staff may still change the order's lines,
// addresses and totals (see ErrSalesOrderNotEditable). source is the
// order's origin (COM-15).
func (o SalesOrder) Editable(source SalesOrderSource) error {
	if source != SalesOrderSourceManual || o.Fulfilled {
		return ErrSalesOrderNotEditable
	}
	switch o.Status {
	case SalesOrderStatusPending, SalesOrderStatusProcessing, SalesOrderStatusOnHold:
		return nil
	default:
		return ErrSalesOrderNotEditable
	}
}
