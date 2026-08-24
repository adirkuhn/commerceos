package reports

import (
	"context"
	"fmt"
	"time"
)

// ProductLifecycleStage classifies a SKU's sales trajectory by comparing
// [From, To) against the immediately preceding period of equal length --
// the Phase 0 "Product Lifecycle" report.
type ProductLifecycleStage string

const (
	// LifecycleNew sold in the current period but not the previous one.
	LifecycleNew ProductLifecycleStage = "new"
	// LifecycleGrowing sold more in the current period than the previous one.
	LifecycleGrowing ProductLifecycleStage = "growing"
	// LifecycleStable sold the same amount in both periods.
	LifecycleStable ProductLifecycleStage = "stable"
	// LifecycleDeclining sold less in the current period than the previous one.
	LifecycleDeclining ProductLifecycleStage = "declining"
	// LifecycleDormant sold in the previous period but not the current one.
	LifecycleDormant ProductLifecycleStage = "dormant"
)

// ProductLifecycleRow is one SKU's current-vs-previous-period comparison
// plus its classified stage.
type ProductLifecycleRow struct {
	SKUID                     *string
	SKUCode                   string
	ProductName               string
	CurrentQuantitySold       int64
	PreviousQuantitySold      int64
	CurrentRevenueMinorUnits  int64
	PreviousRevenueMinorUnits int64
	Stage                     ProductLifecycleStage
}

// ProductLifecycleReport is the Phase 0 "Product Lifecycle" report --
// every SKU that sold in the current or previous period, classified by
// how its sales changed between the two.
type ProductLifecycleReport struct {
	CurrencyCode string
	Rows         []ProductLifecycleRow
}

// GetProductLifecycleReportUseCase is pure composition over the Product
// Performance report, called twice (current + previous period) -- no new
// query, same pattern as GetProductPerformanceMatrixUseCase.
type GetProductLifecycleReportUseCase struct {
	productPerformance productPerformanceReporter
}

func NewGetProductLifecycleReportUseCase(productPerformance productPerformanceReporter) *GetProductLifecycleReportUseCase {
	return &GetProductLifecycleReportUseCase{productPerformance: productPerformance}
}

func (uc *GetProductLifecycleReportUseCase) Execute(ctx context.Context, from, to time.Time) (*ProductLifecycleReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getProductLifecycleReport: %w", ErrToBeforeFrom)
	}
	duration := to.Sub(from)
	prevFrom := from.Add(-duration)
	prevTo := from

	current, err := uc.productPerformance.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getProductLifecycleReport: %w", err)
	}
	previous, err := uc.productPerformance.Execute(ctx, prevFrom, prevTo)
	if err != nil {
		return nil, fmt.Errorf("getProductLifecycleReport: %w", err)
	}

	bySKU := make(map[string]*ProductLifecycleRow)
	var order []string
	get := func(row ProductPerformanceRow) *ProductLifecycleRow {
		key := lifecycleKey(row)
		r, ok := bySKU[key]
		if !ok {
			r = &ProductLifecycleRow{SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName}
			bySKU[key] = r
			order = append(order, key)
		}
		return r
	}
	for _, row := range current.Rows {
		r := get(row)
		r.CurrentQuantitySold = row.QuantitySold
		r.CurrentRevenueMinorUnits = row.RevenueMinorUnits
	}
	for _, row := range previous.Rows {
		r := get(row)
		r.PreviousQuantitySold = row.QuantitySold
		r.PreviousRevenueMinorUnits = row.RevenueMinorUnits
	}

	report := &ProductLifecycleReport{CurrencyCode: current.CurrencyCode}
	if report.CurrencyCode == "" {
		report.CurrencyCode = previous.CurrencyCode
	}
	for _, key := range order {
		r := bySKU[key]
		r.Stage = classifyLifecycleStage(r.CurrentQuantitySold, r.PreviousQuantitySold)
		report.Rows = append(report.Rows, *r)
	}
	return report, nil
}

func lifecycleKey(row ProductPerformanceRow) string {
	if row.SKUID != nil {
		return "id:" + *row.SKUID
	}
	return "code:" + row.SKUCode
}

func classifyLifecycleStage(currentQty, previousQty int64) ProductLifecycleStage {
	switch {
	case previousQty == 0 && currentQty > 0:
		return LifecycleNew
	case previousQty > 0 && currentQty == 0:
		return LifecycleDormant
	case currentQty > previousQty:
		return LifecycleGrowing
	case currentQty < previousQty:
		return LifecycleDeclining
	default:
		return LifecycleStable
	}
}
