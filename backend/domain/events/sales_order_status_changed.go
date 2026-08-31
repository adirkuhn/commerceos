package events

// SalesOrderStatusChanged is raised whenever a SalesOrder's Status field
// actually transitions (never on creation, never a no-op write of the same
// value already there) -- see application/oms.UpdateSalesOrderUseCase,
// which is the one place Status ever changes, whatever the caller (a
// manual edit, a bulk update, or ImportWooOrdersUseCase routing a Woo
// order's own status change back through here). Consumers (see
// infrastructure/consumers.SalesOrderStatusChangedConsumer) re-fetch the
// order's *current* Source/Status to decide whether/how to push it to that
// order's own channel -- the event carries only the id, not a status
// snapshot, since a snapshot could be stale by the time it's processed
// (same convention as ProductChanged).
type SalesOrderStatusChanged struct {
	Base
	TenantID     string `json:"tenant_id"`
	SalesOrderID string `json:"sales_order_id"`
}

func (SalesOrderStatusChanged) EventName() string { return "oms.sales_order.status_changed" }
