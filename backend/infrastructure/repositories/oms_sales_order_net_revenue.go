package repositories

// Revenue reports count what the customer actually kept: every order in a
// revenue-counting status (omsDomain.SalesOrderStatus.CountsAsRevenue),
// net of its refunded lines. Refunds live on the line item
// (refunded_quantity / refunded_amount_minor_units, accumulated by
// RefundLineItemUseCase) -- for a partially_refunded order, and also for an
// order whose refund still awaits manual review (status unchanged), so the
// netting can't key off status. These fragments are the only place that
// arithmetic is written; every report query concatenates them in.
const (
	// netLineQuantitySQL is a line's units sold, net of refunds (alias li).
	netLineQuantitySQL = `(li.quantity - li.refunded_quantity)`
	// netLineRevenueSQL is a line's revenue, net of refunds (alias li).
	netLineRevenueSQL = `(li.total_minor_units - li.refunded_amount_minor_units)`
	// lineNotFullyRefundedSQL drops a fully refunded line from line-level
	// reports -- the SKU didn't sell in that order after all, so it must
	// not count toward the SKU's order count or basket pairs.
	lineNotFullyRefundedSQL = `li.refunded_quantity < li.quantity`
)

// netOrderRevenueSQL is an order's revenue net of its lines' refunds, for
// order-level queries (the order total also carries shipping/fees, which a
// line refund doesn't touch). orderRef is the sales_orders alias or table
// name in the calling query.
func netOrderRevenueSQL(orderRef string) string {
	return `(` + orderRef + `.total_minor_units - COALESCE((SELECT SUM(rl.refunded_amount_minor_units) FROM sales_order_line_items rl WHERE rl.sales_order_id = ` + orderRef + `.id), 0))`
}
