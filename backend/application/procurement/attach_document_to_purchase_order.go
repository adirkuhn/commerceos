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

type AttachDocumentToPurchaseOrderInput struct {
	PurchaseOrderID string
	FileName        string
	ContentType     string
	Data            io.Reader
}

// AttachDocumentToPurchaseOrderUseCase saves an uploaded PDF/CSV/XLSX,
// records it against the PO, and enqueues a DocumentParsingJob for
// cmd/scheduler to pick up -- same "save outside the tx, then a
// tenant.WithTenant block for the DB rows" shape as pms's UploadMediaUseCase.
// The document and job rows are written in the same transaction so a
// document can never exist without a job to process it.
type AttachDocumentToPurchaseOrderUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
	jobRepo      procurementDomain.DocumentParsingJobRepository
	storage      pmsDomain.FileStorage
}

func NewAttachDocumentToPurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
	jobRepo procurementDomain.DocumentParsingJobRepository,
	storage pmsDomain.FileStorage,
) *AttachDocumentToPurchaseOrderUseCase {
	return &AttachDocumentToPurchaseOrderUseCase{db: db, poRepo: poRepo, documentRepo: documentRepo, jobRepo: jobRepo, storage: storage}
}

func (uc *AttachDocumentToPurchaseOrderUseCase) Execute(ctx context.Context, in AttachDocumentToPurchaseOrderInput) (*procurementDomain.DocumentParsingJob, error) {
	var tenantID string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.poRepo.FindByID(ctx, in.PurchaseOrderID); err != nil {
			return err
		}
		tenantID = tenant.IDFromContext(ctx)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("attachDocumentToPurchaseOrder: %w", err)
	}

	url, err := uc.storage.Save(ctx, tenantID, in.FileName, in.Data)
	if err != nil {
		return nil, fmt.Errorf("attachDocumentToPurchaseOrder: %w", err)
	}

	var job procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		doc := procurementDomain.PurchaseOrderDocument{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID,
			FileURL: url, FileName: in.FileName, ContentType: in.ContentType, CreatedAt: now,
		}
		if err := uc.documentRepo.Create(ctx, &doc); err != nil {
			return err
		}
		job = procurementDomain.DocumentParsingJob{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID, DocumentID: doc.ID,
			Status: procurementDomain.DocumentParsingJobPending, CreatedAt: now,
		}
		return uc.jobRepo.Create(ctx, &job)
	})
	if err != nil {
		return nil, fmt.Errorf("attachDocumentToPurchaseOrder: %w", err)
	}
	return &job, nil
}
