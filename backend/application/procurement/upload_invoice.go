package procurement

import (
	"context"
	"fmt"
	"io"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UploadInvoiceInput struct {
	PurchaseOrderID string
	FileName        string
	ContentType     string
	Data            io.Reader
}

// UploadInvoiceUseCase saves a supplier invoice (PDF/CSV/XLSX) against a PO
// and enqueues a ReceivingParsingJob -- same "save outside the tx, then one
// tenant.WithTenant block for the DB rows" shape as
// AttachDocumentToPurchaseOrderUseCase, so the record and its job are
// always created atomically.
type UploadInvoiceUseCase struct {
	db      *gorm.DB
	poRepo  procurementDomain.PurchaseOrderRepository
	recRepo procurementDomain.ReceivingRecordRepository
	jobRepo procurementDomain.ReceivingParsingJobRepository
	storage pmsDomain.FileStorage
}

func NewUploadInvoiceUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	recRepo procurementDomain.ReceivingRecordRepository,
	jobRepo procurementDomain.ReceivingParsingJobRepository,
	storage pmsDomain.FileStorage,
) *UploadInvoiceUseCase {
	return &UploadInvoiceUseCase{db: db, poRepo: poRepo, recRepo: recRepo, jobRepo: jobRepo, storage: storage}
}

func (uc *UploadInvoiceUseCase) Execute(ctx context.Context, in UploadInvoiceInput) (*procurementDomain.ReceivingParsingJob, error) {
	var tenantID string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.poRepo.FindByID(ctx, in.PurchaseOrderID); err != nil {
			return err
		}
		tenantID = tenant.IDFromContext(ctx)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}

	url, err := uc.storage.Save(ctx, tenantID, in.FileName, in.Data)
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}

	var job procurementDomain.ReceivingParsingJob
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		rec := procurementDomain.ReceivingRecord{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID,
			InvoiceFileURL: url, InvoiceFileName: in.FileName, ContentType: in.ContentType,
			Status: procurementDomain.ReceivingRecordStatusPendingReview, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.recRepo.Create(ctx, &rec); err != nil {
			return err
		}
		job = procurementDomain.ReceivingParsingJob{
			TenantID: tenant.IDFromContext(ctx), ReceivingRecordID: rec.ID,
			Status: procurementDomain.ReceivingParsingJobPending, CreatedAt: now,
		}
		return uc.jobRepo.Create(ctx, &job)
	})
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}
	return &job, nil
}
