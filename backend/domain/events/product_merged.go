package events

// ProductMerged is raised after a duplicate product was merged into an
// existing one (see domain/pms.ProductMerge): every reference to each
// duplicate SKU now points at its matched existing SKU, and the duplicate
// product is deleted (its own pms.product.deleted is raised alongside).
type ProductMerged struct {
	Base
	TenantID        string             `json:"tenant_id"`
	SourceProductID string             `json:"source_product_id"`
	TargetProductID string             `json:"target_product_id"`
	SKUs            []ProductMergedSKU `json:"skus"`
}

type ProductMergedSKU struct {
	SourceSKUID string `json:"source_sku_id"`
	TargetSKUID string `json:"target_sku_id"`
}

func (ProductMerged) EventName() string { return "pms.product.merged" }
