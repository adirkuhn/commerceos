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
	RetryCount      int `gorm:"column:retry_count;not null;default:0"`
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

func (r *DocumentParsingJobRepo) RecoverOrphanedRunning(ctx context.Context, maxRetries int, errMsg string, finishedAt time.Time) (int64, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, 0, err
	}
	requeued := tx.Model(&documentParsingJobModel{}).
		Where("status = ? AND retry_count < ?", string(procurementDomain.DocumentParsingJobRunning), maxRetries).
		Updates(map[string]any{
			"status": string(procurementDomain.DocumentParsingJobPending), "retry_count": gorm.Expr("retry_count + 1"), "started_at": nil,
		})
	if requeued.Error != nil {
		return 0, 0, fmt.Errorf("documentParsingJobRepo.RecoverOrphanedRunning: requeue: %w", requeued.Error)
	}
	failed := tx.Model(&documentParsingJobModel{}).Where("status = ?", string(procurementDomain.DocumentParsingJobRunning)).
		Updates(map[string]any{"status": string(procurementDomain.DocumentParsingJobFailed), "error_message": errMsg, "finished_at": finishedAt})
	if failed.Error != nil {
		return requeued.RowsAffected, 0, fmt.Errorf("documentParsingJobRepo.RecoverOrphanedRunning: fail: %w", failed.Error)
	}
	return requeued.RowsAffected, failed.RowsAffected, nil
}

func toParsingJobDomain(m documentParsingJobModel) *procurementDomain.DocumentParsingJob {
	return &procurementDomain.DocumentParsingJob{
		ID: m.ID, TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID, DocumentID: m.DocumentID,
		Status: procurementDomain.DocumentParsingJobStatus(m.Status), ErrorMessage: m.ErrorMessage,
		CreatedAt: m.CreatedAt, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt, RetryCount: m.RetryCount,
	}
}
