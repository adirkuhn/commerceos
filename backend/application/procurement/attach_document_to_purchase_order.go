package procurement

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	// Buffered (not streamed) so the same bytes can be hashed for duplicate
	// detection and then handed to storage -- same tradeoff
	// UploadInvoiceUseCase makes; documents are small enough (PDF/CSV/XLSX)
	// that this isn't a memory concern.
	data, err := io.ReadAll(in.Data)
	if err != nil {
		return nil, fmt.Errorf("attachDocumentToPurchaseOrder: %w", err)
	}
	contentHash := sha256.Sum256(data)
	contentHashHex := hex.EncodeToString(contentHash[:])

	url, err := uc.storage.Save(ctx, tenantID, in.FileName, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("attachDocumentToPurchaseOrder: %w", err)
	}

	var job procurementDomain.DocumentParsingJob
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		duplicates, err := uc.documentRepo.FindDuplicateCandidates(ctx, contentHashHex, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		now := time.Now()
		doc := procurementDomain.PurchaseOrderDocument{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID,
			FileURL: url, FileName: in.FileName, ContentType: in.ContentType,
			ContentHash: contentHashHex, IsPotentialDuplicate: len(duplicates) > 0, CreatedAt: now,
		}
		if err := uc.documentRepo.Create(ctx, &doc); err != nil {
			return err
		}

		// A duplicate on this same PO (e.g. the same file re-attached) only
		// flags the document -- the PO itself is only a "duplicate PO" when
		// the match belongs to a genuinely different PO.
		for _, d := range duplicates {
			if d.PurchaseOrderID != in.PurchaseOrderID {
				if err := uc.poRepo.MarkPotentialDuplicate(ctx, in.PurchaseOrderID, d.PurchaseOrderID); err != nil {
					return err
				}
				break
			}
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
