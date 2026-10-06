package procurement

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	outbox  eventrecord.Enqueuer
}

func NewUploadInvoiceUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	recRepo procurementDomain.ReceivingRecordRepository,
	jobRepo procurementDomain.ReceivingParsingJobRepository,
	storage pmsDomain.FileStorage,
	outbox eventrecord.Enqueuer,
) *UploadInvoiceUseCase {
	return &UploadInvoiceUseCase{db: db, poRepo: poRepo, recRepo: recRepo, jobRepo: jobRepo, storage: storage, outbox: outbox}
}

func (uc *UploadInvoiceUseCase) Execute(ctx context.Context, in UploadInvoiceInput) (*procurementDomain.ReceivingParsingJob, error) {
	var tenantID string
	var supplierID *string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		po, err := uc.poRepo.FindByID(ctx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		supplierID = po.SupplierID
		tenantID = tenant.IDFromContext(ctx)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}

	// Buffered (not streamed) so the same bytes can be hashed for duplicate
	// detection and then handed to storage -- invoices are small enough
	// (PDF/CSV/XLSX) that this isn't a memory concern.
	data, err := io.ReadAll(in.Data)
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}
	contentHash := sha256.Sum256(data)
	contentHashHex := hex.EncodeToString(contentHash[:])

	url, err := uc.storage.Save(ctx, tenantID, in.FileName, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}

	var job procurementDomain.ReceivingParsingJob
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		duplicates, err := uc.recRepo.FindDuplicateCandidates(ctx, contentHashHex, in.PurchaseOrderID, supplierID)
		if err != nil {
			return err
		}
		now := time.Now()
		rec := procurementDomain.ReceivingRecord{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID,
			InvoiceFileURL: url, InvoiceFileName: in.FileName, ContentType: in.ContentType,
			Status:      procurementDomain.ReceivingRecordStatusPendingReview,
			ContentHash: contentHashHex, IsPotentialDuplicate: len(duplicates) > 0,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.recRepo.Create(ctx, &rec); err != nil {
			return err
		}
		job = procurementDomain.ReceivingParsingJob{
			TenantID: tenant.IDFromContext(ctx), ReceivingRecordID: rec.ID,
			Status: procurementDomain.ReceivingParsingJobPending, CreatedAt: now,
		}
		if err := uc.jobRepo.Create(ctx, &job); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ReceivingRecordCreated, rec.ID, eventrecord.Data{
			"purchase_order_id": in.PurchaseOrderID, "file_name": rec.InvoiceFileName, "content_type": rec.ContentType,
			"parsing_job_id": job.ID, "potential_duplicate": rec.IsPotentialDuplicate,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("uploadInvoice: %w", err)
	}
	return &job, nil
}
