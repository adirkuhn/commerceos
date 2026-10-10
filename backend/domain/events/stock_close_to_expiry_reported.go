package events

// StockCloseToExpiryReported is one report of the daily stock-close-to-
// expiry job: the stock with MinDaysLeft..WindowDays days left before it
// expires, for one of the tenant's chosen windows (see
// domain/settings.ExpiryAlert). The Message Center's "stock close to
// expiry" rule turns it into an email -- items is what the template's
// {{#items}} loop repeats over, values pre-formatted for display.
type StockCloseToExpiryReported struct {
	Base
	TenantID    string                   `json:"tenant_id"`
	WindowDays  int                      `json:"window_days"`
	MinDaysLeft int                      `json:"min_days_left"`
	ItemCount   int                      `json:"item_count"`
	Items       []StockCloseToExpiryItem `json:"items"`
}

func (StockCloseToExpiryReported) EventName() string { return "inventory.stock.close_to_expiry_reported" }

// StockCloseToExpiryItem is one SKU's stock sharing one expiry date (summed
// across batches and warehouses).
type StockCloseToExpiryItem struct {
	SKUID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
	ImageURL    string `json:"image_url"`
	Quantity    int64  `json:"quantity"`
	ExpiryDate  string `json:"expiry_date"`
	DaysLeft    int    `json:"days_left"`
	Price       string `json:"price"`
	// SalePrice is "" when the SKU has no running sale.
	SalePrice string `json:"sale_price"`
}
