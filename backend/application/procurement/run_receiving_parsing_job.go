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

// expiryMismatchWindow is how far in the future a received expiry date can
// be without flagging -- an invoice line with no expiry, or one comfortably
// beyond this window, is not worth a human's attention; anything already
// expired or expiring imminently is.
const expiryMismatchWindowDays = 30

// RunReceivingParsingJobUseCase does the parse-then-diff work for one
// ReceivingParsingJob. Unlike RunDocumentParsingJobUseCase (which matches
// extracted lines against the product catalog), this diffs extracted
// invoice lines against the PO's *own* line items -- the PO is the
// expectation, the invoice is reality, and any gap becomes a
// ReceivingDiscrepancy for a human to look at. It does not touch the job's
// own status; cmd/scheduler wraps Execute with MarkRunning/MarkCompleted/
// MarkFailed, same shape as RunDocumentParsingJobUseCase.
type RunReceivingParsingJobUseCase struct {
	db              *gorm.DB
	recRepo         procurementDomain.ReceivingRecordRepository
	lineItemRepo    procurementDomain.PurchaseOrderLineItemRepository
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository
	storage         pmsDomain.FileStorage
	pdfExtractor    procurementDomain.PDFTextExtractor
	// aiParserResolver mirrors RunDocumentParsingJobUseCase's field -- see
	// its doc comment.
	aiParserResolver procurementDomain.AIParserResolver
}

func NewRunReceivingParsingJobUseCase(
	db *gorm.DB,
	recRepo procurementDomain.ReceivingRecordRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository,
	storage pmsDomain.FileStorage,
	pdfExtractor procurementDomain.PDFTextExtractor,
	aiParserResolver procurementDomain.AIParserResolver,
) *RunReceivingParsingJobUseCase {
	return &RunReceivingParsingJobUseCase{
		db: db, recRepo: recRepo, lineItemRepo: lineItemRepo,
		discrepancyRepo: discrepancyRepo, storage: storage, pdfExtractor: pdfExtractor, aiParserResolver: aiParserResolver,
	}
}

// Execute returns the number of discrepancies found, alongside any error --
// cmd/scheduler uses it to pick the completion notification type, same as
// RunDocumentParsingJobUseCase's hasUnmatched bool.
func (uc *RunReceivingParsingJobUseCase) Execute(ctx context.Context, job procurementDomain.ReceivingParsingJob) (int, error) {
	var rec *procurementDomain.ReceivingRecord
	var lineItems []procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		rec, err = uc.recRepo.FindByID(ctx, job.ReceivingRecordID)
		if err != nil {
			return err
		}
		lineItems, err = uc.lineItemRepo.ListByPurchaseOrder(ctx, rec.PurchaseOrderID)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("runReceivingParsingJob: %w", err)
	}

	rc, err := uc.storage.Get(ctx, rec.InvoiceFileURL)
	if err != nil {
		return 0, fmt.Errorf("runReceivingParsingJob: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return 0, fmt.Errorf("runReceivingParsingJob: %w", err)
	}

	extracted, err := extractLineItemsFromDocument(ctx, uc.pdfExtractor, uc.aiParserResolver, rec.ContentType, rec.InvoiceFileName, data)
	if err != nil {
		return 0, fmt.Errorf("runReceivingParsingJob: %w", err)
	}

	discrepancies := diffInvoiceAgainstPurchaseOrder(lineItems, extracted.LineItems)

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		tenantID := tenant.IDFromContext(ctx)
		for i := range discrepancies {
			discrepancies[i].TenantID = tenantID
			discrepancies[i].ReceivingRecordID = rec.ID
			if err := uc.discrepancyRepo.Create(ctx, &discrepancies[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("runReceivingParsingJob: %w", err)
	}
	return len(discrepancies), nil
}

// diffInvoiceAgainstPurchaseOrder matches invoice lines to PO line items via
// NormalizeRawKey (the same raw-text key used by the catalog-matching
// pipeline in match_extracted_line_items.go) and reports every gap: PO
// lines the invoice never mentions (missing_item), invoice lines that don't
// correspond to any PO line (unexpected_item), quantity differences on
// matched lines (quantity_mismatch), and a received expiry date that's
// already passed or imminent (expiry_mismatch). It's a pure function so the
// matching logic itself is unit-testable without a DB.
func diffInvoiceAgainstPurchaseOrder(lineItems []procurementDomain.PurchaseOrderLineItem, extracted []procurementDomain.ExtractedLineItem) []procurementDomain.ReceivingDiscrepancy {
	byKey := make(map[string]*procurementDomain.PurchaseOrderLineItem, len(lineItems))
	for i := range lineItems {
		key := NormalizeRawKey(lineItems[i].RawSKU, lineItems[i].RawDescription)
		if key != "" {
			byKey[key] = &lineItems[i]
		}
	}

	var discrepancies []procurementDomain.ReceivingDiscrepancy
	matched := make(map[string]bool, len(lineItems))
	now := time.Now()

	for _, item := range extracted {
		key := NormalizeRawKey(item.RawSKU, item.RawDescription)
		li, ok := byKey[key]
		if !ok || key == "" {
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				Type: procurementDomain.DiscrepancyTypeUnexpectedItem, ExpectedQty: 0, ReceivedQty: item.Quantity,
				Details: fmt.Sprintf("invoice line %q was not on the purchase order", item.RawDescription),
			})
			continue
		}
		matched[key] = true

		if li.Quantity != item.Quantity {
			lineItemID := li.ID
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeQuantityMismatch,
				ExpectedQty: li.Quantity, ReceivedQty: item.Quantity,
				Details: fmt.Sprintf("expected %d, invoice shows %d", li.Quantity, item.Quantity),
			})
		}

		if item.ExpiryDate != nil && item.ExpiryDate.Before(now.AddDate(0, 0, expiryMismatchWindowDays)) {
			lineItemID := li.ID
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeExpiryMismatch,
				ExpectedQty: li.Quantity, ReceivedQty: item.Quantity,
				Details: fmt.Sprintf("expiry date %s is within %d days (or already past)", item.ExpiryDate.Format("2006-01-02"), expiryMismatchWindowDays),
			})
		}
	}

	for key, li := range byKey {
		if matched[key] {
			continue
		}
		lineItemID := li.ID
		discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
			LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeMissingItem,
			ExpectedQty: li.Quantity, ReceivedQty: 0,
			Details: fmt.Sprintf("%q was on the purchase order but not on the invoice", li.RawDescription),
		})
	}

	return discrepancies
}
