package events

// ProductTranslated is raised when a product's translated name/description
// change. Channel syncs are not affected (they push the default language),
// so it is its own event rather than ProductChanged.
type ProductTranslated struct {
	Base
	TenantID  string `json:"tenant_id"`
	ProductID string `json:"product_id"`
}

func (ProductTranslated) EventName() string { return "pms.product.translated" }
