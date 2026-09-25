package oms

// SalesOrderStatuses lists every SalesOrderStatus -- the universe the
// derived status sets below (and PickableStatuses) are filtered from.
func SalesOrderStatuses() []SalesOrderStatus {
	return []SalesOrderStatus{
		SalesOrderStatusPending, SalesOrderStatusProcessing, SalesOrderStatusOnHold,
		SalesOrderStatusShipped, SalesOrderStatusCompleted, SalesOrderStatusCancelled,
		SalesOrderStatusRefunded, SalesOrderStatusFailed, SalesOrderStatusPartiallyRefunded,
	}
}

// CountsAsRevenue reports whether an order in this status is a real sale
// for reporting: every status that took stock (ReducesStock), plus
// PartiallyRefunded -- the customer kept and paid for the rest of it.
// Revenue reports count every such order net of its refunded lines
// (SalesOrderLineItem.RefundedQuantity/RefundedAmountMinorUnits), so a
// partial refund subtracts what was given back rather than dropping the
// whole order. Pending/cancelled/failed never took stock, and a fully
// refunded order returned all its money.
func (s SalesOrderStatus) CountsAsRevenue() bool {
	return s.ReducesStock() || s == SalesOrderStatusPartiallyRefunded
}

// RevenueCountingStatuses lists every status CountsAsRevenue accepts -- for
// repositories that filter in SQL, so reports can never drift from the rule.
func RevenueCountingStatuses() []SalesOrderStatus {
	var out []SalesOrderStatus
	for _, s := range SalesOrderStatuses() {
		if s.CountsAsRevenue() {
			out = append(out, s)
		}
	}
	return out
}
