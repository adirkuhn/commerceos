package procurement_test

import (
	"bytes"
	"testing"
	"time"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestParseStructuredDocument_CSV_extractsLineItems(t *testing.T) {
	csvData := "SKU,Description,Quantity,Unit Cost\n" +
		"WID-001,Widget A,10,5.00\n" +
		"WID-002,Widget B,3,12.50\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500,
	}, items[0])
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-002", RawDescription: "Widget B", Quantity: 3, UnitCostMinorUnits: 1250,
	}, items[1])
}

func TestParseStructuredDocument_CSV_extractsExpiryDateWhenPresent(t *testing.T) {
	csvData := "SKU,Description,Quantity,Unit Cost,Expiry Date\n" +
		"WID-001,Widget A,10,5.00,2026-12-31\n" +
		"WID-002,Widget B,3,12.50,\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "invoice.csv", []byte(csvData))

	require.NoError(t, err)
	require.Len(t, items, 2)
	require.NotNil(t, items[0].ExpiryDate)
	assert.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), *items[0].ExpiryDate)
	assert.Nil(t, items[1].ExpiryDate, "blank expiry cell must not produce a zero-value date")
}

func TestParseStructuredDocument_CSV_recognizesAlternateDateFormats(t *testing.T) {
	csvData := "SKU,Description,Quantity,Expiry\n" +
		"WID-001,Widget A,10,12/31/2026\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "invoice.csv", []byte(csvData))

	require.NoError(t, err)
	require.NotNil(t, items[0].ExpiryDate)
	assert.Equal(t, 2026, items[0].ExpiryDate.Year())
}

func TestParseStructuredDocument_CSV_skipsBlankRows(t *testing.T) {
	csvData := "SKU,Description,Quantity,Unit Cost\n" +
		"WID-001,Widget A,10,5.00\n" +
		",,,\n" +
		"WID-002,Widget B,3,12.50\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestParseStructuredDocument_CSV_skipsRowsWithInvalidQuantity(t *testing.T) {
	csvData := "SKU,Description,Quantity,Unit Cost\n" +
		"WID-001,Widget A,ten,5.00\n" +
		"WID-002,Widget B,0,5.00\n" +
		"WID-003,Widget C,3,12.50\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "WID-003", items[0].RawSKU)
}

func TestParseStructuredDocument_CSV_allRowsInvalidQuantityReturnsError(t *testing.T) {
	csvData := "SKU,Description,Quantity,Unit Cost\n" +
		"WID-001,Widget A,ten,5.00\n" +
		"WID-002,Widget B,0,5.00\n"

	_, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	assert.ErrorIs(t, err, procurementDomain.ErrNoLineItemsExtracted,
		"a quantity column that was detected but never parses must fail loudly, not silently return zero items")
}

func TestParseStructuredDocument_CSV_missingUnitCostDefaultsToZero(t *testing.T) {
	csvData := "SKU,Description,Quantity\n" +
		"WID-001,Widget A,10\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, int64(0), items[0].UnitCostMinorUnits)
}

func TestParseStructuredDocument_CSV_recognizesColumnSynonymsCaseInsensitively(t *testing.T) {
	csvData := "barcode,item name,qty,price\n" +
		"WID-001,Widget A,10,5.00\n"

	items, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "WID-001", items[0].RawSKU)
	assert.Equal(t, "Widget A", items[0].RawDescription)
}

func TestParseStructuredDocument_CSV_missingRequiredColumnsReturnsError(t *testing.T) {
	csvData := "Color,Size\nRed,Large\n"

	_, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(csvData))

	assert.ErrorIs(t, err, procurementDomain.ErrMissingRequiredColumns)
}

func TestParseStructuredDocument_CSV_emptyFileReturnsError(t *testing.T) {
	_, err := appprocurement.ParseStructuredDocument("text/csv", "order.csv", []byte(""))

	assert.ErrorIs(t, err, procurementDomain.ErrMissingRequiredColumns)
}

func TestParseStructuredDocument_XLSX_extractsLineItems(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	rows := [][]string{
		{"SKU", "Description", "Quantity", "Unit Cost"},
		{"WID-001", "Widget A", "10", "5.00"},
		{"WID-002", "Widget B", "3", "12.50"},
	}
	for r, row := range rows {
		for c, val := range row {
			cellRef, _ := excelize.CoordinatesToCellName(c+1, r+1)
			require.NoError(t, f.SetCellValue(sheet, cellRef, val))
		}
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	items, err := appprocurement.ParseStructuredDocument(
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "order.xlsx", buf.Bytes(),
	)

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500,
	}, items[0])
}

func TestParseStructuredDocument_unsupportedFileTypeReturnsError(t *testing.T) {
	_, err := appprocurement.ParseStructuredDocument("application/pdf", "order.pdf", []byte("not a csv"))

	assert.ErrorIs(t, err, procurementDomain.ErrUnsupportedFileType)
}
