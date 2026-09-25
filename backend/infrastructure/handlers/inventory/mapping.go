package inventory

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

// ── Domain → generated-type mappers ──────────────────────────────────────────

func toGenWarehouse(w invDomain.Warehouse) geninventory.Warehouse {
	return geninventory.Warehouse{
		Id:        w.ID,
		TenantId:  w.TenantID,
		Name:      w.Name,
		Address1:  w.Address1,
		Address2:  w.Address2,
		City:      w.City,
		Province:  w.Province,
		Zip:       w.Zip,
		Country:   w.Country,
		Phone:     w.Phone,
		Active:    w.Active,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func toGenBatch(b invDomain.Batch) geninventory.Batch {
	return geninventory.Batch{
		Id: b.ID, TenantId: b.TenantID, SkuId: b.SKUID, WarehouseId: b.WarehouseID,
		LotNumber: b.LotNumber, ExpiryDate: timePtrToGenDate(b.ExpiryDate),
		ReceivedAt: b.ReceivedAt, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		Quantity: b.Quantity, IsGeneralLedger: b.IsGeneralLedger,
		PurchaseOrderLineItemId: b.PurchaseOrderLineItemID, LandedCostMinorUnits: b.LandedCostMinorUnits,
	}
}

func toGenExpiringBatch(b invDomain.ExpiringBatch) geninventory.ExpiringBatch {
	return geninventory.ExpiringBatch{
		BatchId: b.BatchID, SkuId: b.SKUID, SkuCode: b.SKUCode, ProductName: b.ProductName,
		WarehouseId: b.WarehouseID, WarehouseName: b.WarehouseName, LotNumber: b.LotNumber,
		ExpiryDate: b.ExpiryDate, Quantity: b.Quantity,
	}
}

func toGenExpiredBatchReview(b invDomain.ExpiredBatchReview) geninventory.ExpiredBatchReview {
	return geninventory.ExpiredBatchReview{
		BatchId: b.BatchID, SkuId: b.SKUID, SkuCode: b.SKUCode, ProductName: b.ProductName,
		WarehouseId: b.WarehouseID, WarehouseName: b.WarehouseName, LotNumber: b.LotNumber,
		ExpiryDate: b.ExpiryDate, Quantity: b.Quantity,
	}
}

func toGenStockMovement(m invDomain.StockMovement) geninventory.StockMovement {
	return geninventory.StockMovement{
		Id: m.ID, TenantId: m.TenantID, SkuId: m.SKUID, WarehouseId: m.WarehouseID, BatchId: m.BatchID,
		Type: geninventory.MovementType(m.Type), QuantityDelta: m.QuantityDelta,
		Reason: m.Reason, OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt,
	}
}

func toGenDiscardRecord(d invDomain.DiscardRecord) geninventory.DiscardRecord {
	return geninventory.DiscardRecord{
		Id: d.ID, TenantId: d.TenantID, SkuId: d.SKUID, WarehouseId: d.WarehouseID, BatchId: d.BatchID,
		MovementId: d.MovementID, Type: geninventory.DiscardRecordType(d.Type), Quantity: d.Quantity, Reason: d.Reason, PhotoUrl: d.PhotoURL,
		DiscardedBy: d.DiscardedBy, DiscardedAt: d.DiscardedAt, CreatedAt: d.CreatedAt,
	}
}

func toGenDiscardRecordReport(d invDomain.DiscardRecordReport) geninventory.DiscardRecordReport {
	return geninventory.DiscardRecordReport{
		Id: d.ID, TenantId: d.TenantID, SkuId: d.SKUID, SkuCode: d.SKUCode, ProductName: d.ProductName,
		WarehouseId: d.WarehouseID, WarehouseName: d.WarehouseName, BatchId: d.BatchID, LotNumber: d.LotNumber,
		MovementId: d.MovementID, Type: geninventory.DiscardRecordReportType(d.Type), Quantity: d.Quantity, Reason: d.Reason, PhotoUrl: d.PhotoURL,
		DiscardedBy: d.DiscardedBy, DiscardedAt: d.DiscardedAt, CreatedAt: d.CreatedAt,
	}
}

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

func timeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
