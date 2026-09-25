package procurement

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// genDateToTimePtr/timePtrToGenDate convert between openapi_types.Date
// (date-only, no time-of-day) and *time.Time -- same helpers duplicated in
// infrastructure/handlers/inventory and .../pms; kept local rather than
// shared since none of these handler packages otherwise depend on each other.
func genDateToTimePtr(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

func timePtrToGenDate(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

func lineTypeOr(t *genprocurement.LineItemType) procurementDomain.LineItemType {
	if t == nil {
		return ""
	}
	return procurementDomain.LineItemType(*t)
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// strPtrOrNil is strPtrOr's inverse -- empty string becomes nil, for
// mapping a domain string field back onto an omitempty response field.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// int64PtrOr unwraps an optional integer request field to 0 -- the
// use-case layer treats 0 as "not specified, use the default" for fields
// like UnitsPerPurchasedItem (see AddPurchaseOrderLineItemUseCase).
func int64PtrOr(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// stringPtrOr unwraps an optional string request field to "" -- e.g.
// FlagDamagedLineItemRequest.Note.
func stringPtrOr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// ── Mappers + error handling ─────────────────────────────────────────────────

func toGenPurchaseOrder(po procurementDomain.PurchaseOrder) genprocurement.PurchaseOrder {
	return genprocurement.PurchaseOrder{
		Id: po.ID, TenantId: po.TenantID, SupplierId: po.SupplierID, WarehouseId: po.WarehouseID,
		Status: genprocurement.PurchaseOrderStatus(po.Status), CurrencyCode: po.CurrencyCode,
		ExpectedDate: timePtrToGenDate(po.ExpectedDate), Notes: po.Notes,
		SuggestedSupplierName: &po.SuggestedSupplierName, SuggestedCurrencyCode: &po.SuggestedCurrencyCode,
		IsPotentialDuplicate: po.IsPotentialDuplicate, DuplicateOfPurchaseOrderId: po.DuplicateOfPurchaseOrderID,
		TotalMinorUnits: po.TotalMinorUnits,
		CreatedAt:       po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
}

func toGenLineItem(li procurementDomain.PurchaseOrderLineItem) genprocurement.PurchaseOrderLineItem {
	// LineType's zero value ("") means product (see IsProduct) but isn't
	// itself a valid LineItemType enum value on the wire -- substitute the
	// explicit value so the response always satisfies the schema.
	lineType := li.LineType
	if lineType == "" {
		lineType = procurementDomain.LineItemTypeProduct
	}
	return genprocurement.PurchaseOrderLineItem{
		Id: li.ID, TenantId: li.TenantID, PurchaseOrderId: li.PurchaseOrderID, SkuId: li.SKUID, RawSku: li.RawSKU,
		RawDescription: li.RawDescription, Position: li.Position, ProductName: li.ProductName, SkuCode: li.SKUCode, ProductId: li.ProductID,
		Quantity: li.Quantity, UnitsPerPurchasedItem: li.UnitsPerPurchasedItem, UnitCostMinorUnits: li.UnitCostMinorUnits,
		LineType: genprocurement.LineItemType(lineType), LineTypeConfirmed: li.LineTypeConfirmed,
		SuggestedLotNumber: &li.SuggestedLotNumber, SuggestedExpiryDate: timePtrToGenDate(li.SuggestedExpiryDate),
		CreatedAt: li.CreatedAt, UpdatedAt: li.UpdatedAt,
	}
}

func toGenDocument(d procurementDomain.PurchaseOrderDocument) genprocurement.PurchaseOrderDocument {
	return genprocurement.PurchaseOrderDocument{
		Id: d.ID, PurchaseOrderId: d.PurchaseOrderID, FileUrl: d.FileURL, FileName: d.FileName,
		ContentType: d.ContentType, IsPotentialDuplicate: d.IsPotentialDuplicate, CreatedAt: d.CreatedAt,
	}
}

func toGenDocumentParsingJob(job procurementDomain.DocumentParsingJob) genprocurement.DocumentParsingJob {
	return genprocurement.DocumentParsingJob{
		Id: job.ID, TenantId: job.TenantID, PurchaseOrderId: job.PurchaseOrderID, DocumentId: job.DocumentID,
		Status: genprocurement.DocumentParsingJobStatus(job.Status), ErrorMessage: job.ErrorMessage,
		CreatedAt: job.CreatedAt, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
}

func toGenReceivingRecord(r procurementDomain.ReceivingRecord) genprocurement.ReceivingRecord {
	return genprocurement.ReceivingRecord{
		Id: r.ID, TenantId: r.TenantID, PurchaseOrderId: r.PurchaseOrderID,
		InvoiceFileUrl: r.InvoiceFileURL, InvoiceFileName: r.InvoiceFileName, ContentType: r.ContentType,
		Status: genprocurement.ReceivingRecordStatus(r.Status), IsPotentialDuplicate: r.IsPotentialDuplicate,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toGenReceivingDiscrepancy(d procurementDomain.ReceivingDiscrepancy) genprocurement.ReceivingDiscrepancy {
	return genprocurement.ReceivingDiscrepancy{
		Id: d.ID, TenantId: d.TenantID, ReceivingRecordId: d.ReceivingRecordID, LineItemId: d.LineItemID,
		Type: genprocurement.DiscrepancyType(d.Type), ExpectedQty: d.ExpectedQty, ReceivedQty: d.ReceivedQty,
		Details: d.Details, Status: genprocurement.DiscrepancyStatus(d.Status), AcknowledgedAt: d.AcknowledgedAt,
		CreatedAt: d.CreatedAt,
	}
}

func toGenReceivingParsingJob(job procurementDomain.ReceivingParsingJob) genprocurement.ReceivingParsingJob {
	return genprocurement.ReceivingParsingJob{
		Id: job.ID, TenantId: job.TenantID, ReceivingRecordId: job.ReceivingRecordID,
		Status: genprocurement.ReceivingParsingJobStatus(job.Status), ErrorMessage: job.ErrorMessage,
		CreatedAt: job.CreatedAt, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
}

func toGenBooking(b procurementDomain.Booking) genprocurement.Booking {
	return genprocurement.Booking{
		Id: b.ID, TenantId: b.TenantID, PurchaseOrderId: b.PurchaseOrderID, Status: genprocurement.BookingStatus(b.Status),
		StartedAt: b.StartedAt, PausedAt: b.PausedAt, CompletedAt: b.CompletedAt, CompletedBy: b.CompletedBy,
	}
}

func toGenBookingScan(s procurementDomain.BookingScan) genprocurement.BookingScan {
	return genprocurement.BookingScan{
		Id: s.ID, TenantId: s.TenantID, BookingId: s.BookingID, SkuId: s.SKUID, BatchId: s.BatchID,
		LotNumber: strPtrOrNil(s.LotNumber), ExpiryDate: timePtrToGenDate(s.ExpiryDate),
		ActionType: genprocurement.BookingScanActionType(s.ActionType), Quantity: s.Quantity,
		WriteOffQuantity: s.WriteOffQuantity, Reason: strPtrOrNil(s.Reason),
		ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
}
