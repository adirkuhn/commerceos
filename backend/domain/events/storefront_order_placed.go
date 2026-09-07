package events

// StorefrontOrderPlaced is raised once by CheckoutCartUseCase, right after
// the oms.SalesOrder it creates commits -- the first "an order was created"
// event this codebase has ever raised (SalesOrderStatusChanged fires only
// on a later status transition, never on creation). Carries only the id,
// same minimal-payload convention as SalesOrderStatusChanged: a consumer
// re-fetches the order's current state rather than trusting a payload that
// could already be stale by the time it's processed.
type StorefrontOrderPlaced struct {
	Base
	TenantID     string `json:"tenant_id"`
	SalesOrderID string `json:"sales_order_id"`
}

func (StorefrontOrderPlaced) EventName() string { return "storefront.order.placed" }
