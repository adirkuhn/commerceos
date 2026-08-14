package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM models ──────────────────────────────────────────────────────────────

type receivingRecordModel struct {
	ID                   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             string `gorm:"not null;type:uuid"`
	PurchaseOrderID      string `gorm:"column:purchase_order_id;not null;type:uuid"`
	InvoiceFileURL       string `gorm:"column:invoice_file_url;not null"`
	InvoiceFileName      string `gorm:"column:invoice_file_name;not null;default:''"`
	ContentType          string `gorm:"column:content_type;not null;default:''"`
	Status               string `gorm:"not null;default:'pending_review'"`
	ContentHash          string `gorm:"column:content_hash;not null;default:''"`
	IsPotentialDuplicate bool   `gorm:"column:is_potential_duplicate;not null;default:false"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (receivingRecordModel) TableName() string { return "receiving_records" }

type receivingParsingJobModel struct {
	ID                string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID          string `gorm:"not null;type:uuid"`
	ReceivingRecordID string `gorm:"column:receiving_record_id;not null;type:uuid"`
	Status            string `gorm:"not null;default:'pending'"`
	ErrorMessage      string `gorm:"column:error_message;not null;default:''"`
	CreatedAt         time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	RetryCount        int `gorm:"column:retry_count;not null;default:0"`
}

func (receivingParsingJobModel) TableName() string { return "receiving_parsing_jobs" }

type receivingDiscrepancyModel struct {
	ID                string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID          string  `gorm:"not null;type:uuid"`
	ReceivingRecordID string  `gorm:"column:receiving_record_id;not null;type:uuid"`
	LineItemID        *string `gorm:"column:line_item_id"`
	Type              string  `gorm:"not null"`
	ExpectedQty       int64   `gorm:"column:expected_qty;not null;default:0"`
	ReceivedQty       int64   `gorm:"column:received_qty;not null;default:0"`
	Details           string  `gorm:"not null;default:''"`
	CreatedAt         time.Time
}

func (receivingDiscrepancyModel) TableName() string { return "receiving_discrepancies" }

// ── ReceivingRecordRepo ──────────────────────────────────────────────────────

type ReceivingRecordRepo struct{ db *gorm.DB }

func NewReceivingRecordRepo(db *gorm.DB) *ReceivingRecordRepo { return &ReceivingRecordRepo{db: db} }

func (r *ReceivingRecordRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ReceivingRecordRepo) Create(ctx context.Context, rec *procurementDomain.ReceivingRecord) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := receivingRecordModel{
		TenantID: rec.TenantID, PurchaseOrderID: rec.PurchaseOrderID,
		InvoiceFileURL: rec.InvoiceFileURL, InvoiceFileName: rec.InvoiceFileName, ContentType: rec.ContentType,
		Status: string(rec.Status), ContentHash: rec.ContentHash, IsPotentialDuplicate: rec.IsPotentialDuplicate,
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("receivingRecordRepo.Create: %w", err)
	}
	rec.ID = m.ID
	return nil
}

func (r *ReceivingRecordRepo) FindByID(ctx context.Context, id string) (*procurementDomain.ReceivingRecord, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m receivingRecordModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrReceivingRecordNotFound
		}
		return nil, fmt.Errorf("receivingRecordRepo.FindByID: %w", err)
	}
	return toReceivingRecordDomain(m), nil
}

func (r *ReceivingRecordRepo) ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]procurementDomain.ReceivingRecord, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []receivingRecordModel
	if err := tx.Where("purchase_order_id = ?", purchaseOrderID).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("receivingRecordRepo.ListByPurchaseOrder: %w", err)
	}
	out := make([]procurementDomain.ReceivingRecord, len(rows))
	for i, row := range rows {
		out[i] = *toReceivingRecordDomain(row)
	}
	return out, nil
}

func (r *ReceivingRecordRepo) Update(ctx context.Context, rec *procurementDomain.ReceivingRecord) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&receivingRecordModel{}).Where("id = ?", rec.ID).
		Updates(map[string]any{"status": string(rec.Status), "updated_at": rec.UpdatedAt}).Error; err != nil {
		return fmt.Errorf("receivingRecordRepo.Update: %w", err)
	}
	return nil
}

func toReceivingRecordDomain(m receivingRecordModel) *procurementDomain.ReceivingRecord {
	return &procurementDomain.ReceivingRecord{
		ID: m.ID, TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID,
		InvoiceFileURL: m.InvoiceFileURL, InvoiceFileName: m.InvoiceFileName, ContentType: m.ContentType,
		Status:      procurementDomain.ReceivingRecordStatus(m.Status),
		ContentHash: m.ContentHash, IsPotentialDuplicate: m.IsPotentialDuplicate,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// FindDuplicateCandidates matches ReceivingRecordRepository's doc comment:
// same contentHash, scoped to purchaseOrderID itself or (when supplierID is
// non-nil) any other PO belonging to that supplier -- the join covers the
// second half, the OR covers the first without needing one for a nil
// supplier.
func (r *ReceivingRecordRepo) FindDuplicateCandidates(ctx context.Context, contentHash, purchaseOrderID string, supplierID *string) ([]procurementDomain.ReceivingRecord, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Table("receiving_records AS rr").
		Joins("JOIN purchase_orders AS po ON po.id = rr.purchase_order_id").
		Where("rr.content_hash = ? AND rr.content_hash != ''", contentHash)
	if supplierID != nil {
		q = q.Where("(rr.purchase_order_id = ? OR po.supplier_id = ?)", purchaseOrderID, *supplierID)
	} else {
		q = q.Where("rr.purchase_order_id = ?", purchaseOrderID)
	}
	var rows []receivingRecordModel
	if err := q.Select("rr.*").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("receivingRecordRepo.FindDuplicateCandidates: %w", err)
	}
	out := make([]procurementDomain.ReceivingRecord, len(rows))
	for i, row := range rows {
		out[i] = *toReceivingRecordDomain(row)
	}
	return out, nil
}

// ── ReceivingDiscrepancyRepo ─────────────────────────────────────────────────

type ReceivingDiscrepancyRepo struct{ db *gorm.DB }

func NewReceivingDiscrepancyRepo(db *gorm.DB) *ReceivingDiscrepancyRepo {
	return &ReceivingDiscrepancyRepo{db: db}
}

func (r *ReceivingDiscrepancyRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ReceivingDiscrepancyRepo) Create(ctx context.Context, d *procurementDomain.ReceivingDiscrepancy) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := receivingDiscrepancyModel{
		TenantID: d.TenantID, ReceivingRecordID: d.ReceivingRecordID, LineItemID: d.LineItemID,
		Type: string(d.Type), ExpectedQty: d.ExpectedQty, ReceivedQty: d.ReceivedQty,
		Details: d.Details, CreatedAt: d.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("receivingDiscrepancyRepo.Create: %w", err)
	}
	d.ID = m.ID
	return nil
}

func (r *ReceivingDiscrepancyRepo) ListByReceivingRecord(ctx context.Context, receivingRecordID string) ([]procurementDomain.ReceivingDiscrepancy, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []receivingDiscrepancyModel
	if err := tx.Where("receiving_record_id = ?", receivingRecordID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("receivingDiscrepancyRepo.ListByReceivingRecord: %w", err)
	}
	out := make([]procurementDomain.ReceivingDiscrepancy, len(rows))
	for i, row := range rows {
		out[i] = procurementDomain.ReceivingDiscrepancy{
			ID: row.ID, TenantID: row.TenantID, ReceivingRecordID: row.ReceivingRecordID, LineItemID: row.LineItemID,
			Type: procurementDomain.DiscrepancyType(row.Type), ExpectedQty: row.ExpectedQty, ReceivedQty: row.ReceivedQty,
			Details: row.Details, CreatedAt: row.CreatedAt,
		}
	}
	return out, nil
}

// ── ReceivingParsingJobRepo ──────────────────────────────────────────────────

type ReceivingParsingJobRepo struct{ db *gorm.DB }

func NewReceivingParsingJobRepo(db *gorm.DB) *ReceivingParsingJobRepo {
	return &ReceivingParsingJobRepo{db: db}
}

func (r *ReceivingParsingJobRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ReceivingParsingJobRepo) Create(ctx context.Context, job *procurementDomain.ReceivingParsingJob) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := receivingParsingJobModel{
		TenantID: job.TenantID, ReceivingRecordID: job.ReceivingRecordID,
		Status: string(job.Status), ErrorMessage: job.ErrorMessage, CreatedAt: job.CreatedAt,
		StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("receivingParsingJobRepo.Create: %w", err)
	}
	job.ID = m.ID
	return nil
}

func (r *ReceivingParsingJobRepo) FindByID(ctx context.Context, id string) (*procurementDomain.ReceivingParsingJob, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m receivingParsingJobModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrParsingJobNotFound
		}
		return nil, fmt.Errorf("receivingParsingJobRepo.FindByID: %w", err)
	}
	return toReceivingParsingJobDomain(m), nil
}

func (r *ReceivingParsingJobRepo) ListPending(ctx context.Context) ([]procurementDomain.ReceivingParsingJob, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []receivingParsingJobModel
	if err := tx.Where("status = ?", string(procurementDomain.ReceivingParsingJobPending)).
		Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("receivingParsingJobRepo.ListPending: %w", err)
	}
	out := make([]procurementDomain.ReceivingParsingJob, len(rows))
	for i, row := range rows {
		out[i] = *toReceivingParsingJobDomain(row)
	}
	return out, nil
}

func (r *ReceivingParsingJobRepo) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&receivingParsingJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(procurementDomain.ReceivingParsingJobRunning), "started_at": startedAt}).Error; err != nil {
		return fmt.Errorf("receivingParsingJobRepo.MarkRunning: %w", err)
	}
	return nil
}

func (r *ReceivingParsingJobRepo) MarkCompleted(ctx context.Context, id string, finishedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&receivingParsingJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(procurementDomain.ReceivingParsingJobCompleted), "finished_at": finishedAt}).Error; err != nil {
		return fmt.Errorf("receivingParsingJobRepo.MarkCompleted: %w", err)
	}
	return nil
}

func (r *ReceivingParsingJobRepo) MarkFailed(ctx context.Context, id string, errMsg string, finishedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&receivingParsingJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(procurementDomain.ReceivingParsingJobFailed), "error_message": errMsg, "finished_at": finishedAt}).Error; err != nil {
		return fmt.Errorf("receivingParsingJobRepo.MarkFailed: %w", err)
	}
	return nil
}

func (r *ReceivingParsingJobRepo) RecoverOrphanedRunning(ctx context.Context, maxRetries int, errMsg string, finishedAt time.Time) (int64, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, 0, err
	}
	requeued := tx.Model(&receivingParsingJobModel{}).
		Where("status = ? AND retry_count < ?", string(procurementDomain.ReceivingParsingJobRunning), maxRetries).
		Updates(map[string]any{
			"status": string(procurementDomain.ReceivingParsingJobPending), "retry_count": gorm.Expr("retry_count + 1"), "started_at": nil,
		})
	if requeued.Error != nil {
		return 0, 0, fmt.Errorf("receivingParsingJobRepo.RecoverOrphanedRunning: requeue: %w", requeued.Error)
	}
	failed := tx.Model(&receivingParsingJobModel{}).Where("status = ?", string(procurementDomain.ReceivingParsingJobRunning)).
		Updates(map[string]any{"status": string(procurementDomain.ReceivingParsingJobFailed), "error_message": errMsg, "finished_at": finishedAt})
	if failed.Error != nil {
		return requeued.RowsAffected, 0, fmt.Errorf("receivingParsingJobRepo.RecoverOrphanedRunning: fail: %w", failed.Error)
	}
	return requeued.RowsAffected, failed.RowsAffected, nil
}

func toReceivingParsingJobDomain(m receivingParsingJobModel) *procurementDomain.ReceivingParsingJob {
	return &procurementDomain.ReceivingParsingJob{
		ID: m.ID, TenantID: m.TenantID, ReceivingRecordID: m.ReceivingRecordID,
		Status: procurementDomain.ReceivingParsingJobStatus(m.Status), ErrorMessage: m.ErrorMessage,
		CreatedAt: m.CreatedAt, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt, RetryCount: m.RetryCount,
	}
}
