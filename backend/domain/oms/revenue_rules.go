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
// for reporting: every status that took stock (ReducesStock) -- a status
// that never took stock (pending, cancelled, failed) was never a sale, and
// a fully refunded order returned all its money.
func (s SalesOrderStatus) CountsAsRevenue() bool {
	return s.ReducesStock()
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
