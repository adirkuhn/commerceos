package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM models ──────────────────────────────────────────────────────────────

type purchaseOrderModel struct {
	ID                         string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                   string  `gorm:"not null;type:uuid"`
	SupplierID                 *string `gorm:"column:supplier_id;type:uuid"`
	WarehouseID                *string `gorm:"column:warehouse_id;type:uuid"`
	Status                     string  `gorm:"not null;default:'draft'"`
	CurrencyCode               string  `gorm:"not null;default:'USD'"`
	ExpectedDate               *time.Time
	Notes                      string  `gorm:"not null;default:''"`
	SuggestedSupplierName      string  `gorm:"not null;default:''"`
	SuggestedCurrencyCode      string  `gorm:"not null;default:''"`
	IsPotentialDuplicate       bool    `gorm:"column:is_potential_duplicate;not null;default:false"`
	DuplicateOfPurchaseOrderID *string `gorm:"column:duplicate_of_purchase_order_id;type:uuid"`
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

func (purchaseOrderModel) TableName() string { return "purchase_orders" }

type purchaseOrderLineItemModel struct {
	ID                     string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID               string  `gorm:"not null;type:uuid"`
	PurchaseOrderID        string  `gorm:"column:purchase_order_id;not null;type:uuid"`
	SKUID                  *string `gorm:"column:sku_id"`
	RawSKU                 string  `gorm:"column:raw_sku;not null;default:''"`
	RawDescription         string  `gorm:"not null;default:''"`
	Position               int64   `gorm:"column:position;not null;default:0"`
	Quantity               int64   `gorm:"not null"`
	UnitCostMinorUnits     int64   `gorm:"not null;default:0"`
	SKUAutoLinkedFromAlias bool    `gorm:"column:sku_auto_linked_from_alias;not null;default:false"`
	LineType               string  `gorm:"column:line_type;not null;default:''"`
	LineTypeConfirmed      bool    `gorm:"column:line_type_confirmed;not null;default:false"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (purchaseOrderLineItemModel) TableName() string { return "purchase_order_line_items" }

// ── PurchaseOrderRepo ────────────────────────────────────────────────────────

type PurchaseOrderRepo struct{ db *gorm.DB }

func NewPurchaseOrderRepo(db *gorm.DB) *PurchaseOrderRepo { return &PurchaseOrderRepo{db: db} }

func (r *PurchaseOrderRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PurchaseOrderRepo) Create(ctx context.Context, po *procurementDomain.PurchaseOrder) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := purchaseOrderModel{
		TenantID: po.TenantID, SupplierID: po.SupplierID, WarehouseID: po.WarehouseID,
		Status: string(po.Status), CurrencyCode: po.CurrencyCode, ExpectedDate: po.ExpectedDate, Notes: po.Notes,
		CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("purchaseOrderRepo.Create: %w", err)
	}
	po.ID = m.ID
	return nil
}

func (r *PurchaseOrderRepo) FindByID(ctx context.Context, id string) (*procurementDomain.PurchaseOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m purchaseOrderModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrPurchaseOrderNotFound
		}
		return nil, fmt.Errorf("purchaseOrderRepo.FindByID: %w", err)
	}
	return toPurchaseOrderDomain(m), nil
}

func (r *PurchaseOrderRepo) List(ctx context.Context) ([]procurementDomain.PurchaseOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []purchaseOrderModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("purchaseOrderRepo.List: %w", err)
	}
	out := make([]procurementDomain.PurchaseOrder, len(rows))
	for i, row := range rows {
		out[i] = *toPurchaseOrderDomain(row)
	}
	return out, nil
}

func (r *PurchaseOrderRepo) Update(ctx context.Context, po *procurementDomain.PurchaseOrder) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&purchaseOrderModel{}).Where("id = ?", po.ID).
		Updates(map[string]any{
			"supplier_id":   po.SupplierID,
			"warehouse_id":  po.WarehouseID,
			"status":        string(po.Status),
			"currency_code": po.CurrencyCode,
			"expected_date": po.ExpectedDate,
			"notes":         po.Notes,
			"updated_at":    po.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("purchaseOrderRepo.Update: %w", err)
	}
	return nil
}

func (r *PurchaseOrderRepo) UpdateSuggestions(ctx context.Context, id string, supplierName, currencyCode string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&purchaseOrderModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"suggested_supplier_name": supplierName,
			"suggested_currency_code": currencyCode,
		}).Error; err != nil {
		return fmt.Errorf("purchaseOrderRepo.UpdateSuggestions: %w", err)
	}
	return nil
}

func (r *PurchaseOrderRepo) MarkPotentialDuplicate(ctx context.Context, id, duplicateOfPurchaseOrderID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&purchaseOrderModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"is_potential_duplicate":         true,
			"duplicate_of_purchase_order_id": duplicateOfPurchaseOrderID,
		}).Error; err != nil {
		return fmt.Errorf("purchaseOrderRepo.MarkPotentialDuplicate: %w", err)
	}
	return nil
}

func (r *PurchaseOrderRepo) ClearPotentialDuplicate(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&purchaseOrderModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"is_potential_duplicate":         false,
			"duplicate_of_purchase_order_id": nil,
		}).Error; err != nil {
		return fmt.Errorf("purchaseOrderRepo.ClearPotentialDuplicate: %w", err)
	}
	return nil
}

func (r *PurchaseOrderRepo) FindDuplicatesOf(ctx context.Context, id string) ([]procurementDomain.PurchaseOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []purchaseOrderModel
	if err := tx.Where("duplicate_of_purchase_order_id = ?", id).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("purchaseOrderRepo.FindDuplicatesOf: %w", err)
	}
	out := make([]procurementDomain.PurchaseOrder, len(rows))
	for i, row := range rows {
		out[i] = *toPurchaseOrderDomain(row)
	}
	return out, nil
}

func (r *PurchaseOrderRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&purchaseOrderModel{}).Error; err != nil {
		return fmt.Errorf("purchaseOrderRepo.Delete: %w", err)
	}
	return nil
}

func toPurchaseOrderDomain(m purchaseOrderModel) *procurementDomain.PurchaseOrder {
	return &procurementDomain.PurchaseOrder{
		ID: m.ID, TenantID: m.TenantID, SupplierID: m.SupplierID, WarehouseID: m.WarehouseID,
		Status: procurementDomain.PurchaseOrderStatus(m.Status), CurrencyCode: m.CurrencyCode,
		ExpectedDate: m.ExpectedDate, Notes: m.Notes,
		SuggestedSupplierName: m.SuggestedSupplierName, SuggestedCurrencyCode: m.SuggestedCurrencyCode,
		IsPotentialDuplicate: m.IsPotentialDuplicate, DuplicateOfPurchaseOrderID: m.DuplicateOfPurchaseOrderID,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// ── PurchaseOrderLineItemRepo ────────────────────────────────────────────────

type PurchaseOrderLineItemRepo struct{ db *gorm.DB }

func NewPurchaseOrderLineItemRepo(db *gorm.DB) *PurchaseOrderLineItemRepo {
	return &PurchaseOrderLineItemRepo{db: db}
}

func (r *PurchaseOrderLineItemRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PurchaseOrderLineItemRepo) Create(ctx context.Context, li *procurementDomain.PurchaseOrderLineItem) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := purchaseOrderLineItemModel{
		TenantID: li.TenantID, PurchaseOrderID: li.PurchaseOrderID, SKUID: li.SKUID, RawSKU: li.RawSKU,
		RawDescription: li.RawDescription, Position: li.Position, Quantity: li.Quantity, UnitCostMinorUnits: li.UnitCostMinorUnits,
		SKUAutoLinkedFromAlias: li.SKUAutoLinkedFromAlias,
		LineType:               string(li.LineType),
		LineTypeConfirmed:      li.LineTypeConfirmed,
		CreatedAt:              li.CreatedAt, UpdatedAt: li.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("purchaseOrderLineItemRepo.Create: %w", err)
	}
	li.ID = m.ID
	return nil
}

func (r *PurchaseOrderLineItemRepo) FindByID(ctx context.Context, id string) (*procurementDomain.PurchaseOrderLineItem, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m purchaseOrderLineItemModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrLineItemNotFound
		}
		return nil, fmt.Errorf("purchaseOrderLineItemRepo.FindByID: %w", err)
	}
	return toLineItemDomain(m), nil
}

func (r *PurchaseOrderLineItemRepo) ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]procurementDomain.PurchaseOrderLineItem, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []purchaseOrderLineItemModel
	if err := tx.Where("purchase_order_id = ?", purchaseOrderID).Order("position asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("purchaseOrderLineItemRepo.ListByPurchaseOrder: %w", err)
	}
	out := make([]procurementDomain.PurchaseOrderLineItem, len(rows))
	for i, row := range rows {
		out[i] = *toLineItemDomain(row)
	}
	return out, nil
}

func (r *PurchaseOrderLineItemRepo) Update(ctx context.Context, li *procurementDomain.PurchaseOrderLineItem) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&purchaseOrderLineItemModel{}).Where("id = ?", li.ID).
		Updates(map[string]any{
			"sku_id":                     li.SKUID,
			"raw_sku":                    li.RawSKU,
			"raw_description":            li.RawDescription,
			"quantity":                   li.Quantity,
			"unit_cost_minor_units":      li.UnitCostMinorUnits,
			"sku_auto_linked_from_alias": li.SKUAutoLinkedFromAlias,
			"line_type":                  string(li.LineType),
			"line_type_confirmed":        li.LineTypeConfirmed,
			"updated_at":                 li.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("purchaseOrderLineItemRepo.Update: %w", err)
	}
	return nil
}

func (r *PurchaseOrderLineItemRepo) SumByPurchaseOrders(ctx context.Context, purchaseOrderIDs []string) (map[string]int64, error) {
	totals := make(map[string]int64, len(purchaseOrderIDs))
	if len(purchaseOrderIDs) == 0 {
		return totals, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	type sumRow struct {
		PurchaseOrderID string
		Total           int64
	}
	// unit_cost_minor_units is always stored non-negative (see
	// AddLineItemRequest/UpdateLineItemRequest's openapi minimum:0) -- a
	// discount/credit_note row's amount subtracts from the total, so it must
	// be negated here rather than relying on a negative stored value (see
	// procurementDomain.LineItemType.IsReduction, and
	// GetPurchaseOrderUseCase's Go-side equivalent for the single-PO path).
	var rows []sumRow
	if err := tx.Model(&purchaseOrderLineItemModel{}).
		Select("purchase_order_id, SUM(CASE WHEN line_type IN (?, ?) THEN -(quantity * unit_cost_minor_units) ELSE quantity * unit_cost_minor_units END) AS total",
			string(procurementDomain.LineItemTypeDiscount), string(procurementDomain.LineItemTypeCreditNote)).
		Where("purchase_order_id IN ?", purchaseOrderIDs).
		Group("purchase_order_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("purchaseOrderLineItemRepo.SumByPurchaseOrders: %w", err)
	}
	for _, row := range rows {
		totals[row.PurchaseOrderID] = row.Total
	}
	return totals, nil
}

func (r *PurchaseOrderLineItemRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&purchaseOrderLineItemModel{}).Error; err != nil {
		return fmt.Errorf("purchaseOrderLineItemRepo.Delete: %w", err)
	}
	return nil
}

func toLineItemDomain(m purchaseOrderLineItemModel) *procurementDomain.PurchaseOrderLineItem {
	return &procurementDomain.PurchaseOrderLineItem{
		ID: m.ID, TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID, SKUID: m.SKUID, RawSKU: m.RawSKU,
		RawDescription: m.RawDescription, Position: m.Position, Quantity: m.Quantity, UnitCostMinorUnits: m.UnitCostMinorUnits,
		SKUAutoLinkedFromAlias: m.SKUAutoLinkedFromAlias,
		LineType:               procurementDomain.LineItemType(m.LineType),
		LineTypeConfirmed:      m.LineTypeConfirmed,
		CreatedAt:              m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// ── PurchaseOrderDocumentRepo ────────────────────────────────────────────────

type purchaseOrderDocumentModel struct {
	ID                   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             string `gorm:"not null;type:uuid"`
	PurchaseOrderID      string `gorm:"column:purchase_order_id;not null;type:uuid"`
	FileURL              string `gorm:"column:file_url;not null"`
	FileName             string `gorm:"column:file_name;not null;default:''"`
	ContentType          string `gorm:"column:content_type;not null;default:''"`
	ContentHash          string `gorm:"column:content_hash;not null;default:''"`
	IsPotentialDuplicate bool   `gorm:"column:is_potential_duplicate;not null;default:false"`
	CreatedAt            time.Time
}

func (purchaseOrderDocumentModel) TableName() string { return "purchase_order_documents" }

type PurchaseOrderDocumentRepo struct{ db *gorm.DB }

func NewPurchaseOrderDocumentRepo(db *gorm.DB) *PurchaseOrderDocumentRepo {
	return &PurchaseOrderDocumentRepo{db: db}
}

func (r *PurchaseOrderDocumentRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PurchaseOrderDocumentRepo) Create(ctx context.Context, doc *procurementDomain.PurchaseOrderDocument) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := purchaseOrderDocumentModel{
		TenantID: doc.TenantID, PurchaseOrderID: doc.PurchaseOrderID, FileURL: doc.FileURL,
		FileName: doc.FileName, ContentType: doc.ContentType,
		ContentHash: doc.ContentHash, IsPotentialDuplicate: doc.IsPotentialDuplicate, CreatedAt: doc.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("purchaseOrderDocumentRepo.Create: %w", err)
	}
	doc.ID = m.ID
	return nil
}

func (r *PurchaseOrderDocumentRepo) FindByID(ctx context.Context, id string) (*procurementDomain.PurchaseOrderDocument, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m purchaseOrderDocumentModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrDocumentNotFound
		}
		return nil, fmt.Errorf("purchaseOrderDocumentRepo.FindByID: %w", err)
	}
	return toDocumentDomain(m), nil
}

func (r *PurchaseOrderDocumentRepo) ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]procurementDomain.PurchaseOrderDocument, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []purchaseOrderDocumentModel
	if err := tx.Where("purchase_order_id = ?", purchaseOrderID).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("purchaseOrderDocumentRepo.ListByPurchaseOrder: %w", err)
	}
	out := make([]procurementDomain.PurchaseOrderDocument, len(rows))
	for i, row := range rows {
		out[i] = *toDocumentDomain(row)
	}
	return out, nil
}

func (r *PurchaseOrderDocumentRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&purchaseOrderDocumentModel{}).Error; err != nil {
		return fmt.Errorf("purchaseOrderDocumentRepo.Delete: %w", err)
	}
	return nil
}

// FindDuplicateCandidates matches ReceivingRecordRepo's method of the same
// name: same contentHash, scoped to purchaseOrderID itself or (when
// supplierID is non-nil) any other PO belonging to that supplier.
func (r *PurchaseOrderDocumentRepo) FindDuplicateCandidates(ctx context.Context, contentHash, _ string) ([]procurementDomain.PurchaseOrderDocument, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	// Tenant-wide (RLS already scopes this to the current tenant) -- not
	// filtered by purchaseOrderID or supplier, since the same file can be
	// uploaded to two supplier-less POs before either has one assigned.
	var rows []purchaseOrderDocumentModel
	if err := tx.Where("content_hash = ? AND content_hash != ''", contentHash).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("purchaseOrderDocumentRepo.FindDuplicateCandidates: %w", err)
	}
	out := make([]procurementDomain.PurchaseOrderDocument, len(rows))
	for i, row := range rows {
		out[i] = *toDocumentDomain(row)
	}
	return out, nil
}

func toDocumentDomain(m purchaseOrderDocumentModel) *procurementDomain.PurchaseOrderDocument {
	return &procurementDomain.PurchaseOrderDocument{
		ID: m.ID, TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID,
		FileURL: m.FileURL, FileName: m.FileName, ContentType: m.ContentType,
		ContentHash: m.ContentHash, IsPotentialDuplicate: m.IsPotentialDuplicate, CreatedAt: m.CreatedAt,
	}
}

// ── DocumentParsingJobRepo ───────────────────────────────────────────────────

type documentParsingJobModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	PurchaseOrderID string `gorm:"column:purchase_order_id;not null;type:uuid"`
	DocumentID      string `gorm:"column:document_id;not null;type:uuid"`
	Status          string `gorm:"not null;default:'pending'"`
	ErrorMessage    string `gorm:"column:error_message;not null;default:''"`
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

func (documentParsingJobModel) TableName() string { return "document_parsing_jobs" }

type DocumentParsingJobRepo struct{ db *gorm.DB }

func NewDocumentParsingJobRepo(db *gorm.DB) *DocumentParsingJobRepo {
	return &DocumentParsingJobRepo{db: db}
}

func (r *DocumentParsingJobRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *DocumentParsingJobRepo) Create(ctx context.Context, job *procurementDomain.DocumentParsingJob) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := documentParsingJobModel{
		TenantID: job.TenantID, PurchaseOrderID: job.PurchaseOrderID, DocumentID: job.DocumentID,
		Status: string(job.Status), ErrorMessage: job.ErrorMessage, CreatedAt: job.CreatedAt,
		StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("documentParsingJobRepo.Create: %w", err)
	}
	job.ID = m.ID
	return nil
}

func (r *DocumentParsingJobRepo) FindByID(ctx context.Context, id string) (*procurementDomain.DocumentParsingJob, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m documentParsingJobModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrParsingJobNotFound
		}
		return nil, fmt.Errorf("documentParsingJobRepo.FindByID: %w", err)
	}
	return toParsingJobDomain(m), nil
}

// ListPending is a system-level, cross-tenant query -- callers must run it
// under tenant.WithSuperAdmin, same as ImportJobRepository.ListPending.
func (r *DocumentParsingJobRepo) ListPending(ctx context.Context) ([]procurementDomain.DocumentParsingJob, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []documentParsingJobModel
	if err := tx.Where("status = ?", string(procurementDomain.DocumentParsingJobPending)).
		Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("documentParsingJobRepo.ListPending: %w", err)
	}
	out := make([]procurementDomain.DocumentParsingJob, len(rows))
	for i, row := range rows {
		out[i] = *toParsingJobDomain(row)
	}
	return out, nil
}

func (r *DocumentParsingJobRepo) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&documentParsingJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(procurementDomain.DocumentParsingJobRunning), "started_at": startedAt}).Error; err != nil {
		return fmt.Errorf("documentParsingJobRepo.MarkRunning: %w", err)
	}
	return nil
}

func (r *DocumentParsingJobRepo) MarkCompleted(ctx context.Context, id string, finishedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&documentParsingJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(procurementDomain.DocumentParsingJobCompleted), "finished_at": finishedAt}).Error; err != nil {
		return fmt.Errorf("documentParsingJobRepo.MarkCompleted: %w", err)
	}
	return nil
}

func (r *DocumentParsingJobRepo) MarkFailed(ctx context.Context, id string, errMsg string, finishedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&documentParsingJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(procurementDomain.DocumentParsingJobFailed), "error_message": errMsg, "finished_at": finishedAt}).Error; err != nil {
		return fmt.Errorf("documentParsingJobRepo.MarkFailed: %w", err)
	}
	return nil
}

func (r *DocumentParsingJobRepo) MarkOrphanedRunningAsFailed(ctx context.Context, errMsg string, finishedAt time.Time) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	result := tx.Model(&documentParsingJobModel{}).Where("status = ?", string(procurementDomain.DocumentParsingJobRunning)).
		Updates(map[string]any{"status": string(procurementDomain.DocumentParsingJobFailed), "error_message": errMsg, "finished_at": finishedAt})
	if result.Error != nil {
		return 0, fmt.Errorf("documentParsingJobRepo.MarkOrphanedRunningAsFailed: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func toParsingJobDomain(m documentParsingJobModel) *procurementDomain.DocumentParsingJob {
	return &procurementDomain.DocumentParsingJob{
		ID: m.ID, TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID, DocumentID: m.DocumentID,
		Status: procurementDomain.DocumentParsingJobStatus(m.Status), ErrorMessage: m.ErrorMessage,
		CreatedAt: m.CreatedAt, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt,
	}
}

// ── SupplierProductAliasRepo ─────────────────────────────────────────────────

type supplierProductAliasModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	SupplierID string `gorm:"column:supplier_id;not null;type:uuid"`
	RawKey     string `gorm:"column:raw_key;not null"`
	SKUID      string `gorm:"column:sku_id;not null;type:uuid"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (supplierProductAliasModel) TableName() string { return "supplier_product_aliases" }

type SupplierProductAliasRepo struct{ db *gorm.DB }

func NewSupplierProductAliasRepo(db *gorm.DB) *SupplierProductAliasRepo {
	return &SupplierProductAliasRepo{db: db}
}

func (r *SupplierProductAliasRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert writes or updates the (tenant, supplier, raw_key) alias -- relies
// on the supplier_product_aliases_unique constraint from migration 027.
func (r *SupplierProductAliasRepo) Upsert(ctx context.Context, alias *procurementDomain.SupplierProductAlias) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := supplierProductAliasModel{
		TenantID: alias.TenantID, SupplierID: alias.SupplierID, RawKey: alias.RawKey, SKUID: alias.SKUID,
		CreatedAt: alias.CreatedAt, UpdatedAt: alias.UpdatedAt,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "supplier_id"}, {Name: "raw_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"sku_id", "updated_at"}),
	}).Create(&m).Error; err != nil {
		return fmt.Errorf("supplierProductAliasRepo.Upsert: %w", err)
	}
	alias.ID = m.ID
	return nil
}

func (r *SupplierProductAliasRepo) FindBySupplierAndRawKey(ctx context.Context, supplierID, rawKey string) (*procurementDomain.SupplierProductAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m supplierProductAliasModel
	if err := tx.Where("supplier_id = ? AND raw_key = ?", supplierID, rawKey).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("supplierProductAliasRepo.FindBySupplierAndRawKey: %w", err)
	}
	return toAliasDomain(m), nil
}

func toAliasDomain(m supplierProductAliasModel) *procurementDomain.SupplierProductAlias {
	return &procurementDomain.SupplierProductAlias{
		ID: m.ID, TenantID: m.TenantID, SupplierID: m.SupplierID, RawKey: m.RawKey, SKUID: m.SKUID,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
