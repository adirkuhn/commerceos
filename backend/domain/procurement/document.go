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
	// ContentHash/IsPotentialDuplicate mirror ReceivingRecord's duplicate
	// detection (see ReceivingRecordRepository.FindDuplicateCandidates) --
	// IsPotentialDuplicate is set at creation when another document with
	// the same sha256 content hash already exists on this PO or (when the
	// PO has a supplier) any other PO belonging to that supplier.
	ContentHash          string
	IsPotentialDuplicate bool
	CreatedAt            time.Time
}
