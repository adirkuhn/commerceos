// Package pdf adapts a PDF text-extraction library to the
// procurement.PDFTextExtractor port. Kept as its own thin adapter package
// (rather than calling ledongthuc/pdf straight from application code) so the
// underlying library can be swapped later without touching anything above
// the port -- LedongthucExtractor is the only thing that knows which
// library is in use.
package pdf

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

type LedongthucExtractor struct{}

func NewLedongthucExtractor() *LedongthucExtractor {
	return &LedongthucExtractor{}
}

func (e *LedongthucExtractor) ExtractText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("ledongthucExtractor: open: %w", err)
	}
	textReader, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("ledongthucExtractor: extract: %w", err)
	}
	text, err := io.ReadAll(textReader)
	if err != nil {
		return "", fmt.Errorf("ledongthucExtractor: read: %w", err)
	}
	return string(text), nil
}
