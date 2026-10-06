package procurement

import (
	"context"
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
	outbox           eventrecord.Enqueuer
}

func NewRunReceivingParsingJobUseCase(
	db *gorm.DB,
	recRepo procurementDomain.ReceivingRecordRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository,
	storage pmsDomain.FileStorage,
	pdfExtractor procurementDomain.PDFTextExtractor,
	aiParserResolver procurementDomain.AIParserResolver,
	outbox eventrecord.Enqueuer,
) *RunReceivingParsingJobUseCase {
	return &RunReceivingParsingJobUseCase{
		db: db, recRepo: recRepo, lineItemRepo: lineItemRepo,
		discrepancyRepo: discrepancyRepo, storage: storage, pdfExtractor: pdfExtractor, aiParserResolver: aiParserResolver,
		outbox: outbox,
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
			if err := recordDiscrepancyDetected(ctx, uc.outbox, discrepancies[i], DiscrepancySourceInvoice); err != nil {
				return err
			}
		}
		return eventrecord.Record(ctx, uc.outbox, events.ReceivingRecordParsed, rec.ID, eventrecord.Data{
			"purchase_order_id": rec.PurchaseOrderID, "parsing_job_id": job.ID, "discrepancy_count": len(discrepancies),
		})
	})
	if err != nil {
		return 0, fmt.Errorf("runReceivingParsingJob: %w", err)
	}
	return len(discrepancies), nil
}

// diffInvoiceAgainstPurchaseOrder matches invoice lines to PO line items and
// reports every gap: PO lines the invoice never mentions (missing_item),
// invoice lines that don't correspond to any PO line (unexpected_item),
// quantity differences on matched lines (quantity_mismatch), and a received
// expiry date that's already passed or imminent (expiry_mismatch). It's a
// pure function so the matching logic itself is unit-testable without a DB.
//
// Matching is two-tier, not a single exact-key lookup: PO line items and
// invoice line items come from two independent extraction passes (the PO's
// own document parse, and this receiving invoice's parse) that can even be
// the same physical file uploaded twice -- AI/PDF extraction isn't
// byte-deterministic, so RawSKU/RawDescription commonly differ between the
// two passes for the same product (e.g. one pass captures a barcode as
// RawSKU, the other leaves it blank). An exact-string key alone would then
// show the same product as both missing and unexpected. So:
//  1. Try an exact match on normalized RawSKU first (cheap, precise when
//     both sides captured it).
//  2. Otherwise fall back to fuzzy RawDescription matching, reusing the
//     same significantTokens/nameMatchScore pair (and acceptance bar) the
//     Phase-3 catalog-matching pipeline already uses in
//     match_extracted_line_items.go, instead of inventing a second
//     normalization scheme.
//
// Invoice quantities are accumulated per PO line (not compared 1:1 per
// invoice line) before diffing against the ordered quantity, so a supplier
// splitting one PO line across multiple invoice lines by batch/expiry
// (ExtractedLineItem.BatchCode/ExpiryDate) sums correctly instead of
// producing a spurious quantity_mismatch per split line. PO lines are
// tracked individually (never collapsed into a single-key map), so a
// duplicate RawSKU across two PO lines can't cause one to silently vanish
// from the diff.
//
// A genuinely ambiguous item (no SKU match, no fuzzy match clearing the
// bar) is left as an ordinary missing_item/unexpected_item pair -- this
// function never guesses past its threshold. LinkReceivingDiscrepanciesUseCase
// is the deliberate escape hatch for a human to pair those up by hand.
func diffInvoiceAgainstPurchaseOrder(lineItems []procurementDomain.PurchaseOrderLineItem, extracted []procurementDomain.ExtractedLineItem) []procurementDomain.ReceivingDiscrepancy {
	bySKU := make(map[string][]*procurementDomain.PurchaseOrderLineItem, len(lineItems))
	allLines := make([]*procurementDomain.PurchaseOrderLineItem, len(lineItems))
	for i := range lineItems {
		li := &lineItems[i]
		allLines[i] = li
		if key := NormalizeRawKey(li.RawSKU, ""); key != "" {
			bySKU[key] = append(bySKU[key], li)
		}
	}

	receivedQty := make(map[string]int64, len(lineItems))
	var discrepancies []procurementDomain.ReceivingDiscrepancy
	now := time.Now()

	for _, item := range extracted {
		var li *procurementDomain.PurchaseOrderLineItem
		if key := NormalizeRawKey(item.RawSKU, ""); key != "" {
			if candidates := bySKU[key]; len(candidates) > 0 {
				li = pickLeastFilledLineItem(candidates, receivedQty)
			}
		}
		if li == nil {
			li = matchLineItemByDescription(item.RawDescription, allLines)
		}
		if li == nil {
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				Type: procurementDomain.DiscrepancyTypeUnexpectedItem, Status: procurementDomain.DiscrepancyStatusOpen,
				ExpectedQty: 0, ReceivedQty: item.Quantity,
				Details: fmt.Sprintf("invoice line %q was not on the purchase order", item.RawDescription),
			})
			continue
		}
		receivedQty[li.ID] += item.Quantity

		if item.ExpiryDate != nil && item.ExpiryDate.Before(now.AddDate(0, 0, expiryMismatchWindowDays)) {
			lineItemID := li.ID
			expiry := *item.ExpiryDate
			// The date itself lives in ExpiryDate, never in Details: stored
			// text can't follow the tenant's date-format setting.
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeExpiryMismatch, Status: procurementDomain.DiscrepancyStatusOpen,
				ExpectedQty: li.Quantity, ReceivedQty: item.Quantity, ExpiryDate: &expiry,
				Details: fmt.Sprintf("expiry date is within %d days (or already past)", expiryMismatchWindowDays),
			})
		}
	}

	for _, li := range allLines {
		got := receivedQty[li.ID]
		lineItemID := li.ID
		switch {
		case got == 0:
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeMissingItem, Status: procurementDomain.DiscrepancyStatusOpen,
				ExpectedQty: li.Quantity, ReceivedQty: 0,
				Details: fmt.Sprintf("%q was on the purchase order but not on the invoice", li.RawDescription),
			})
		case got != li.Quantity:
			discrepancies = append(discrepancies, procurementDomain.ReceivingDiscrepancy{
				LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeQuantityMismatch, Status: procurementDomain.DiscrepancyStatusOpen,
				ExpectedQty: li.Quantity, ReceivedQty: got,
				Details: fmt.Sprintf("expected %d, invoice shows %d", li.Quantity, got),
			})
		}
	}

	return discrepancies
}

// pickLeastFilledLineItem resolves a duplicate-RawSKU collision: returns the
// first candidate that hasn't yet received its full ordered quantity, or
// candidates[0] if every candidate is already filled (extra invoice qty in
// that case surfaces as a quantity_mismatch overflow on that line, which is
// the correct signal -- the supplier over-shipped, not under-shipped).
func pickLeastFilledLineItem(candidates []*procurementDomain.PurchaseOrderLineItem, receivedQty map[string]int64) *procurementDomain.PurchaseOrderLineItem {
	for _, li := range candidates {
		if receivedQty[li.ID] < li.Quantity {
			return li
		}
	}
	return candidates[0]
}

// matchLineItemByDescription is diffInvoiceAgainstPurchaseOrder's fallback
// when no PO line shares the invoice line's RawSKU -- same acceptance bar
// as matchByName in match_extracted_line_items.go (score >= nameMatchMinScore,
// margin >= nameMatchMinMargin over the runner-up), scored against every PO
// line's RawDescription instead of a catalog product name. Returns nil (not
// an error) when nothing clears both bars.
func matchLineItemByDescription(rawDescription string, candidates []*procurementDomain.PurchaseOrderLineItem) *procurementDomain.PurchaseOrderLineItem {
	tokens := significantTokens(rawDescription)
	if len(tokens) == 0 {
		return nil
	}
	best := -1
	bestScore, secondScore := -1.0, -1.0
	for i, li := range candidates {
		score := nameMatchScore(li.RawDescription, tokens)
		if score > bestScore {
			best, bestScore, secondScore = i, score, bestScore
		} else if score > secondScore {
			secondScore = score
		}
	}
	if best == -1 || bestScore < nameMatchMinScore || bestScore-secondScore < nameMatchMinMargin {
		return nil
	}
	return candidates[best]
}
