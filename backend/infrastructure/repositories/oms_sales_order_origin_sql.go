package repositories

import "fmt"

// SQL fragments that read an order's origin (sales_order_origins, COM-15)
// for queries over sales_orders; table is the alias the query gives
// sales_orders. Every order has exactly one origin row (unique on
// sales_order_id), so these are index lookups.

// salesOrderOriginSQL is the order's origin (pos, manual, woocommerce, ...).
func salesOrderOriginSQL(table string) string {
	return fmt.Sprintf("(SELECT soo.origin FROM sales_order_origins soo WHERE soo.sales_order_id = %s.id)", table)
}

// notPOSSaleSQL: the order is not a POS sale (handed over at the counter,
// never picked, packed or shipped).
func notPOSSaleSQL(table string) string {
	return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM sales_order_origins soo WHERE soo.sales_order_id = %s.id AND soo.origin = 'pos')", table)
}

// fromConnectionSQL: the order was imported through the connection bound
// to its one ? placeholder.
func fromConnectionSQL(table string) string {
	return fmt.Sprintf("%s.id IN (SELECT soo.sales_order_id FROM sales_order_origins soo WHERE soo.connection_id = ?)", table)
}
