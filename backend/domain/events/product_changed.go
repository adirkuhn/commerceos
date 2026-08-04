package events

// ProductChanged is raised whenever a product's core fields or its default
// SKU's price/weight/dimensions change. Consumers (see
// infrastructure/consumers) re-push the product's *current* state to every
// channel it's already linked to — the event carries only the product id,
// not a field snapshot, since a snapshot could be stale by the time it's
// processed.
type ProductChanged struct {
	Base
	TenantID  string `json:"tenant_id"`
	ProductID string `json:"product_id"`
}

func (ProductChanged) EventName() string { return "pms.product.changed" }
