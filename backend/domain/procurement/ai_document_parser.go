package procurement

import "context"

// DocumentInput is the raw bytes of an unstructured (PDF) supplier document
// handed to an AIDocumentParser adapter.
type DocumentInput struct {
	FileName    string
	ContentType string
	Data        []byte
}

// DocumentExtractionResult is what an AIDocumentParser pulls out of an
// unstructured document -- line items plus a best-effort supplier name for
// the review UI to suggest (see PurchaseOrder.SuggestedSupplierName).
// SupplierName is empty when the document didn't have a clear one; there's
// deliberately no deterministic fallback for it (unlike currency, a
// supplier's letterhead has no reliable fixed pattern across arbitrary
// invoice layouts, so guessing wrong would be worse than not guessing).
type DocumentExtractionResult struct {
	LineItems    []ExtractedLineItem
	SupplierName string
}

// AIDocumentParser is a provider-agnostic port for extracting line items
// (and a supplier-name suggestion) from an unstructured document --
// implemented per-vendor in infrastructure/ai/ (OpenAI first; Claude/
// DeepSeek/etc addable later without touching application code). Only
// reached when the deterministic CSV/XLSX path doesn't apply and
// PDFTextExtractor couldn't pull a usable text layer.
type AIDocumentParser interface {
	ExtractLineItems(ctx context.Context, doc DocumentInput) (DocumentExtractionResult, error)
}

// PDFTextExtractor is a swappable port around whichever PDF text-extraction
// library is configured -- kept separate from AIDocumentParser so a PDF with
// a clean text layer can skip the AI call entirely (extract text, then run
// it through the same deterministic matching the CSV/XLSX path uses).
type PDFTextExtractor interface {
	ExtractText(data []byte) (string, error)
}
