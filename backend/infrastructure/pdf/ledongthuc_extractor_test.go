package pdf_test

import (
	"bytes"
	"fmt"
	"testing"

	pdfadapter "github.com/adirkuhn/commerceos/infrastructure/pdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildMinimalPDF hand-assembles the smallest valid single-page PDF
// containing the given text in a content stream, computing the xref byte
// offsets as it writes -- there's no PDF-writing library in this repo
// (excelize/ledongthuc are both read-only), and pulling one in just to
// generate a test fixture would be a much bigger dependency than the ~40
// lines this takes.
func buildMinimalPDF(t *testing.T, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	offsets := make([]int, 0, 5)

	write := func(s string) {
		buf.WriteString(s)
	}
	startObj := func() {
		offsets = append(offsets, buf.Len())
	}

	write("%PDF-1.4\n")

	startObj()
	write("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	startObj()
	write("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	startObj()
	write("3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 612 792] /Contents 5 0 R >>\nendobj\n")

	startObj()
	write("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	startObj()
	content := fmt.Sprintf("BT /F1 24 Tf 100 700 Td (%s) Tj ET", text)
	write(fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content))

	xrefStart := buf.Len()
	write("xref\n0 6\n0000000000 65535 f \n")
	for _, off := range offsets {
		write(fmt.Sprintf("%010d 00000 n \n", off))
	}
	write("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	write(fmt.Sprintf("startxref\n%d\n%%%%EOF", xrefStart))

	return buf.Bytes()
}

func TestLedongthucExtractor_ExtractText_returnsTheDocumentsText(t *testing.T) {
	data := buildMinimalPDF(t, "Hello World")
	extractor := pdfadapter.NewLedongthucExtractor()

	text, err := extractor.ExtractText(data)

	require.NoError(t, err)
	assert.Contains(t, text, "Hello World")
}

func TestLedongthucExtractor_ExtractText_returnsErrorForInvalidPDF(t *testing.T) {
	extractor := pdfadapter.NewLedongthucExtractor()

	_, err := extractor.ExtractText([]byte("this is not a pdf"))

	assert.Error(t, err)
}
