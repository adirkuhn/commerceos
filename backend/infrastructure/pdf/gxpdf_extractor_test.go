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
// (excelize/gxpdf are both read-only), and pulling one in just to generate
// a test fixture would be a much bigger dependency than the ~40 lines this
// takes.
func buildMinimalPDF(t *testing.T, text string) []byte {
	t.Helper()
	return buildMinimalPDFWithHeader(t, "%PDF-1.4\n", text)
}

// buildMinimalPDFWithHeader is buildMinimalPDF with the header line
// swappable -- offsets are computed *after* writing whatever header is
// given, so a non-standard-length header (see the trailing-space
// regression test below) doesn't desync the xref table.
func buildMinimalPDFWithHeader(t *testing.T, header, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	offsets := make([]int, 0, 5)

	write := func(s string) {
		buf.WriteString(s)
	}
	startObj := func() {
		offsets = append(offsets, buf.Len())
	}

	write(header)

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

func TestGxPDFExtractor_ExtractText_returnsTheDocumentsText(t *testing.T) {
	data := buildMinimalPDF(t, "Hello World")
	extractor := pdfadapter.NewGxPDFExtractor()

	text, err := extractor.ExtractText(data)

	require.NoError(t, err)
	assert.Contains(t, text, "Hello World")
}

func TestGxPDFExtractor_ExtractText_returnsErrorForInvalidPDF(t *testing.T) {
	extractor := pdfadapter.NewGxPDFExtractor()

	_, err := extractor.ExtractText([]byte("this is not a pdf"))

	assert.Error(t, err)
}

// This is the actual bug that motivated swapping away from ledongthuc/pdf:
// some real-world PDF generators emit a header like "%PDF-1.7 \n" (a
// trailing space before the newline) -- spec-legal (the header line is a
// comment, free-form after the version), but ledongthuc/pdf's header check
// demanded the byte right after the version be exactly \r or \n and
// rejected the file outright with "not a PDF file: invalid header" even
// though the rest of the document was perfectly well-formed.
func TestGxPDFExtractor_ExtractText_toleratesTrailingSpaceInHeader(t *testing.T) {
	data := buildMinimalPDFWithHeader(t, "%PDF-1.7 \n", "Hello World")
	extractor := pdfadapter.NewGxPDFExtractor()

	text, err := extractor.ExtractText(data)

	require.NoError(t, err)
	assert.Contains(t, text, "Hello World")
}
