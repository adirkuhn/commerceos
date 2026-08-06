package procurement

import (
	"context"
	"strings"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// extractedDocumentInfo is extractLineItemsFromDocument's result --
// SuggestedSupplierName/SuggestedCurrencyCode are always empty for the
// deterministic CSV/XLSX path (a spreadsheet has no letterhead or printed
// currency code to find) and only populated for the PDF path, where
// SuggestedCurrencyCode comes from a cheap deterministic regex pass over
// the extracted text (see DetectCurrencyCode) and SuggestedSupplierName
// from the AI parser (see AIDocumentParser.ExtractLineItems's doc comment
// for why supplier name specifically needs AI, not a regex).
type extractedDocumentInfo struct {
	LineItems             []procurementDomain.ExtractedLineItem
	SuggestedSupplierName string
	SuggestedCurrencyCode string
}

// extractLineItemsFromDocument is the structured-first, AI-fallback parse
// dispatch shared by RunDocumentParsingJobUseCase (PO documents) and
// RunReceivingParsingJobUseCase (supplier invoices) -- both jobs need the
// exact same "try deterministic CSV/XLSX parsing, else PDF text + regex +
// AI" logic, only what happens to the result afterward differs.
func extractLineItemsFromDocument(
	ctx context.Context,
	pdfExtractor procurementDomain.PDFTextExtractor,
	aiParserResolver procurementDomain.AIParserResolver,
	contentType, fileName string,
	data []byte,
) (extractedDocumentInfo, error) {
	if IsStructuredDocument(contentType, fileName) {
		items, err := ParseStructuredDocument(contentType, fileName, data)
		if err != nil {
			return extractedDocumentInfo{}, err
		}
		return extractedDocumentInfo{LineItems: items}, nil
	}

	text, err := pdfExtractor.ExtractText(data)
	if err != nil {
		return extractedDocumentInfo{}, err
	}
	if strings.TrimSpace(text) == "" {
		return extractedDocumentInfo{}, procurementDomain.ErrPDFHasNoExtractableText
	}
	currencyCode, _ := DetectCurrencyCode(text)

	if aiParserResolver == nil {
		return extractedDocumentInfo{}, procurementDomain.ErrAIParsingNotConfigured
	}
	aiParser, err := aiParserResolver.Resolve(ctx)
	if err != nil {
		return extractedDocumentInfo{}, err
	}
	if aiParser == nil {
		return extractedDocumentInfo{}, procurementDomain.ErrAIParsingNotConfigured
	}
	result, err := aiParser.ExtractLineItems(ctx, procurementDomain.DocumentInput{
		FileName: fileName, ContentType: contentType, Data: data, ExtractedText: text,
	})
	if err != nil {
		return extractedDocumentInfo{}, err
	}
	return extractedDocumentInfo{
		LineItems: result.LineItems, SuggestedSupplierName: result.SupplierName, SuggestedCurrencyCode: currencyCode,
	}, nil
}
