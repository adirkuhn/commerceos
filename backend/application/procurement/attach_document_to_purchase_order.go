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
// document can never exist without a job to process it. Exception: when the
// document is a duplicate of one already parsed on another PO, no job is
// enqueued at all -- see maybeCopyFromDuplicate.
type AttachDocumentToPurchaseOrderUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	jobRepo      procurementDomain.DocumentParsingJobRepository
	storage      pmsDomain.FileStorage
}

func NewAttachDocumentToPurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	jobRepo procurementDomain.DocumentParsingJobRepository,
	storage pmsDomain.FileStorage,
) *AttachDocumentToPurchaseOrderUseCase {
	return &AttachDocumentToPurchaseOrderUseCase{
		db: db, poRepo: poRepo, documentRepo: documentRepo, lineItemRepo: lineItemRepo, jobRepo: jobRepo, storage: storage,
	}
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
		var duplicateOfPOID string
		for _, d := range duplicates {
			if d.PurchaseOrderID != in.PurchaseOrderID {
				duplicateOfPOID = d.PurchaseOrderID
				if err := uc.poRepo.MarkPotentialDuplicate(ctx, in.PurchaseOrderID, duplicateOfPOID); err != nil {
					return err
				}
				break
			}
		}

		if duplicateOfPOID != "" {
			copied, err := uc.copyFromDuplicate(ctx, in.PurchaseOrderID, duplicateOfPOID, now)
			if err != nil {
				return err
			}
			if copied {
				job = procurementDomain.DocumentParsingJob{
					TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID, DocumentID: doc.ID,
					Status: procurementDomain.DocumentParsingJobCompleted, CreatedAt: now, StartedAt: &now, FinishedAt: &now,
				}
				return uc.jobRepo.Create(ctx, &job)
			}
			// Falls through to the normal parse below -- the matched PO
			// has no line items yet (its own parse may still be pending,
			// or failed), so there's nothing to copy.
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

// copyFromDuplicate detects a duplicate *before* any parsing happens: if
// the matching PO already has line items (from its own, earlier parse), a
// duplicate is the same document, so there's nothing new to learn from
// running AI/deterministic extraction on it again -- copy its line items
// (and supplier, if this PO doesn't have one yet) directly instead. Returns
// false (no-op) if the matched PO has no line items yet, so the caller
// falls back to a normal parse.
func (uc *AttachDocumentToPurchaseOrderUseCase) copyFromDuplicate(ctx context.Context, purchaseOrderID, duplicateOfPOID string, now time.Time) (bool, error) {
	sourceItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, duplicateOfPOID)
	if err != nil {
		return false, err
	}
	if len(sourceItems) == 0 {
		return false, nil
	}

	po, err := uc.poRepo.FindByID(ctx, purchaseOrderID)
	if err != nil {
		return false, err
	}
	if po.SupplierID == nil {
		source, err := uc.poRepo.FindByID(ctx, duplicateOfPOID)
		if err != nil {
			return false, err
		}
		if source.SupplierID != nil {
			po.SupplierID = source.SupplierID
		}
	}

	for _, item := range sourceItems {
		copy := procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: purchaseOrderID,
			SKUID: item.SKUID, RawSKU: item.RawSKU, RawDescription: item.RawDescription,
			Quantity: item.Quantity, UnitCostMinorUnits: item.UnitCostMinorUnits,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.lineItemRepo.Create(ctx, &copy); err != nil {
			return false, err
		}
	}

	// Same "fully resolved -> back to draft" rule LinkLineItemToSKUUseCase
	// applies -- a duplicate that copied a supplier and every line item
	// already matched to a SKU doesn't need a human to review it too.
	return true, revertPurchaseOrderFromReviewIfFullyResolved(ctx, uc.poRepo, uc.lineItemRepo, po)
}
