// Package pdf adapts a PDF text-extraction library to the
// procurement.PDFTextExtractor port. Kept as its own thin adapter package
// (rather than calling gxpdf straight from application code) so the
// underlying library can be swapped later without touching anything above
// the port -- GxPDFExtractor is the only thing that knows which library is
// in use.
package pdf

import (
	"fmt"

	"github.com/coregx/gxpdf"
)

type GxPDFExtractor struct{}

func NewGxPDFExtractor() *GxPDFExtractor {
	return &GxPDFExtractor{}
}

func (e *GxPDFExtractor) ExtractText(data []byte) (string, error) {
	doc, err := gxpdf.OpenFromBytes(data)
	if err != nil {
		return "", fmt.Errorf("gxpdfExtractor: open: %w", err)
	}
	defer doc.Close()

	var text string
	for _, page := range doc.Pages() {
		text += page.ExtractText()
	}
	return text, nil
}
