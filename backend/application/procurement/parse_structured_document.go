package procurement

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/xuri/excelize/v2"
)

// ParseStructuredDocument extracts line items from a supplier CSV or XLSX
// file with no AI involved -- deterministic header-based column detection,
// per CLAUDE.md's third-party gate: only unstructured PDFs go through the
// AI port (see ai_document_parser.go), structured formats are parsed in
// code because it's cheaper, faster, and doesn't depend on an external API
// being up.
func ParseStructuredDocument(contentType, fileName string, data []byte) ([]procurementDomain.ExtractedLineItem, error) {
	switch {
	case isCSV(contentType, fileName):
		return parseCSVRows(data)
	case isXLSX(contentType, fileName):
		return parseXLSXRows(data)
	default:
		return nil, procurementDomain.ErrUnsupportedFileType
	}
}

func isCSV(contentType, fileName string) bool {
	return strings.Contains(contentType, "csv") || strings.HasSuffix(strings.ToLower(fileName), ".csv")
}

func isXLSX(contentType, fileName string) bool {
	return strings.Contains(contentType, "spreadsheetml") || strings.HasSuffix(strings.ToLower(fileName), ".xlsx")
}

// IsStructuredDocument reports whether ParseStructuredDocument can handle
// this file (CSV/XLSX) -- used by the document-parsing job runner to decide
// between the deterministic path and the PDF/AI path.
func IsStructuredDocument(contentType, fileName string) bool {
	return isCSV(contentType, fileName) || isXLSX(contentType, fileName)
}

func parseCSVRows(data []byte) ([]procurementDomain.ExtractedLineItem, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parseCSVRows: %w", err)
	}
	return rowsToLineItems(rows)
}

func parseXLSXRows(data []byte) ([]procurementDomain.ExtractedLineItem, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parseXLSXRows: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, procurementDomain.ErrMissingRequiredColumns
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("parseXLSXRows: %w", err)
	}
	return rowsToLineItems(rows)
}

// columnIndex maps a recognized column's position in the header row; -1
// means the column wasn't found.
type columnIndex struct {
	sku         int
	description int
	quantity    int
	unitCost    int
	expiryDate  int
}

var columnSynonyms = map[string][]string{
	"sku":         {"sku", "barcode", "code", "item code", "product code"},
	"description": {"description", "name", "item", "product", "item name"},
	"quantity":    {"quantity", "qty"},
	"unitCost":    {"unit cost", "unitcost", "cost", "price", "unit price"},
	"expiryDate":  {"expiry date", "expiry", "expiration date", "expirationdate", "best before"},
}

func detectColumns(header []string) columnIndex {
	idx := columnIndex{sku: -1, description: -1, quantity: -1, unitCost: -1, expiryDate: -1}
	for i, raw := range header {
		h := strings.ToLower(strings.TrimSpace(raw))
		for _, syn := range columnSynonyms["sku"] {
			if h == syn {
				idx.sku = i
			}
		}
		for _, syn := range columnSynonyms["description"] {
			if h == syn {
				idx.description = i
			}
		}
		for _, syn := range columnSynonyms["quantity"] {
			if h == syn {
				idx.quantity = i
			}
		}
		for _, syn := range columnSynonyms["unitCost"] {
			if h == syn {
				idx.unitCost = i
			}
		}
		for _, syn := range columnSynonyms["expiryDate"] {
			if h == syn {
				idx.expiryDate = i
			}
		}
	}
	return idx
}

func rowsToLineItems(rows [][]string) ([]procurementDomain.ExtractedLineItem, error) {
	if len(rows) == 0 {
		return nil, procurementDomain.ErrMissingRequiredColumns
	}
	idx := detectColumns(rows[0])
	if idx.quantity == -1 || (idx.sku == -1 && idx.description == -1) {
		return nil, procurementDomain.ErrMissingRequiredColumns
	}

	var items []procurementDomain.ExtractedLineItem
	for _, row := range rows[1:] {
		if isBlankRow(row) {
			continue
		}
		qty, err := strconv.ParseInt(strings.TrimSpace(cell(row, idx.quantity)), 10, 64)
		if err != nil || qty <= 0 {
			continue
		}
		unitCostMinorUnits := int64(0)
		if idx.unitCost != -1 {
			if cost, err := strconv.ParseFloat(strings.TrimSpace(cell(row, idx.unitCost)), 64); err == nil {
				unitCostMinorUnits = int64(cost*100 + 0.5)
			}
		}
		var expiryDate *time.Time
		if idx.expiryDate != -1 {
			if parsed, err := parseFlexibleDate(cell(row, idx.expiryDate)); err == nil {
				expiryDate = &parsed
			}
		}
		items = append(items, procurementDomain.ExtractedLineItem{
			RawSKU:             cell(row, idx.sku),
			RawDescription:     cell(row, idx.description),
			Quantity:           qty,
			UnitCostMinorUnits: unitCostMinorUnits,
			ExpiryDate:         expiryDate,
		})
	}
	if len(items) == 0 && len(rows) > 1 {
		return nil, procurementDomain.ErrNoLineItemsExtracted
	}
	return items, nil
}

func cell(row []string, i int) string {
	if i == -1 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// parseFlexibleDate tries the handful of date formats suppliers actually
// use in spreadsheets -- ISO first since it's unambiguous, then the two
// common locale orderings.
func parseFlexibleDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "01/02/2006", "02/01/2006", "2006/01/02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}

func isBlankRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
