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

// RunDocumentParsingJobUseCase does the actual parse-match-apply work for
// one DocumentParsingJob. It does not touch the job's own status --
// cmd/scheduler wraps a call to Execute with MarkRunning/MarkCompleted/
// MarkFailed, same shape as its ImportJob runner (see runImportJob).
type RunDocumentParsingJobUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
	matchUC      *MatchExtractedLineItemsUseCase
	storage      pmsDomain.FileStorage
	pdfExtractor procurementDomain.PDFTextExtractor
	// aiParserResolver looks up whichever AIDocumentParser the job's own
	// tenant has configured (see integrations.AIConnection) -- resolving
	// per job instead of once at process startup, since different tenants
	// can each pick their own provider. Resolves to nil when the tenant
	// hasn't configured one, and PDFs then fail clearly with
	// ErrAIParsingNotConfigured instead of the job crashing.
	aiParserResolver procurementDomain.AIParserResolver
}

func NewRunDocumentParsingJobUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
	matchUC *MatchExtractedLineItemsUseCase,
	storage pmsDomain.FileStorage,
	pdfExtractor procurementDomain.PDFTextExtractor,
	aiParserResolver procurementDomain.AIParserResolver,
) *RunDocumentParsingJobUseCase {
	return &RunDocumentParsingJobUseCase{
		db: db, poRepo: poRepo, lineItemRepo: lineItemRepo, documentRepo: documentRepo,
		matchUC: matchUC, storage: storage, pdfExtractor: pdfExtractor, aiParserResolver: aiParserResolver,
	}
}

// Execute returns whether the PO ended up needing human review (any line
// item unmatched), alongside any error -- cmd/scheduler uses it to pick
// between TypeDocumentParsingCompleted and TypePurchaseOrderNeedsReview for
// the completion notification.
func (uc *RunDocumentParsingJobUseCase) Execute(ctx context.Context, job procurementDomain.DocumentParsingJob) (bool, error) {
	var doc *procurementDomain.PurchaseOrderDocument
	var po *procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		doc, err = uc.documentRepo.FindByID(ctx, job.DocumentID)
		if err != nil {
			return err
		}
		po, err = uc.poRepo.FindByID(ctx, job.PurchaseOrderID)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("runDocumentParsingJob: %w", err)
	}

	rc, err := uc.storage.Get(ctx, doc.FileURL)
	if err != nil {
		return false, fmt.Errorf("runDocumentParsingJob: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return false, fmt.Errorf("runDocumentParsingJob: %w", err)
	}

	extracted, err := extractLineItemsFromDocument(ctx, uc.pdfExtractor, uc.aiParserResolver, doc.ContentType, doc.FileName, data)
	if err != nil {
		return false, fmt.Errorf("runDocumentParsingJob: %w", err)
	}

	hasUnmatched := false
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		supplierID := ""
		if po.SupplierID != nil {
			supplierID = *po.SupplierID
		}
		matched, err := uc.matchUC.Execute(ctx, supplierID, extracted.LineItems)
		if err != nil {
			return err
		}

		// Suggestions are informational only (see PurchaseOrder's doc
		// comment) -- only worth writing when there's no supplier yet to
		// suggest a name for; still fine to call with empty strings, that
		// just clears any stale suggestion from a previous parse attempt.
		if po.SupplierID == nil {
			if err := uc.poRepo.UpdateSuggestions(ctx, po.ID, extracted.SuggestedSupplierName, extracted.SuggestedCurrencyCode); err != nil {
				return err
			}
		}

		now := time.Now()
		for i, m := range matched {
			li := procurementDomain.PurchaseOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID, SKUID: m.SKUID,
				RawSKU: m.Extracted.RawSKU, RawDescription: m.Extracted.RawDescription, Position: int64(i),
				Quantity: m.Extracted.Quantity, UnitsPerPurchasedItem: m.UnitsPerPurchasedItem, UnitCostMinorUnits: m.Extracted.UnitCostMinorUnits,
				SKUAutoLinkedFromAlias: m.MatchSource == MatchSourceAlias,
				LineType:               m.Extracted.LineType,
				SuggestedLotNumber:     m.Extracted.BatchCode,
				SuggestedExpiryDate:    m.Extracted.ExpiryDate,
				CreatedAt:              now, UpdatedAt: now,
			}
			if err := uc.lineItemRepo.Create(ctx, &li); err != nil {
				return err
			}
			// A charge row (shipping/discount/fee/credit note) is never
			// "unmatched" -- it was never supposed to have a SKU.
			if m.SKUID == nil && li.LineType.IsProduct() {
				hasUnmatched = true
			}
		}

		// A nil-supplier PO already started life as needs_review (see
		// CreatePurchaseOrderUseCase) -- only a status change here needs
		// writing, which is exactly the hasUnmatched-without-supplier-issue
		// case: draft -> needs_review.
		if hasUnmatched && po.Status != procurementDomain.PurchaseOrderStatusNeedsReview {
			po.Status = procurementDomain.PurchaseOrderStatusNeedsReview
			po.UpdatedAt = now
			return uc.poRepo.Update(ctx, po)
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("runDocumentParsingJob: %w", err)
	}
	return hasUnmatched || po.SupplierID == nil, nil
}
