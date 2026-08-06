package procurement

import "time"

// PurchaseOrderDocument is a supplier-sent file (PDF/CSV/XLSX) attached to a
// PurchaseOrder, uploaded so its line items can be extracted and linked to
// products. Stored via the same pms.FileStorage port used for product media.
type PurchaseOrderDocument struct {
	ID              string
	TenantID        string
	PurchaseOrderID string
	FileURL         string
	FileName        string
	ContentType     string
	CreatedAt       time.Time
}
