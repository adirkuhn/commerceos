package storefront

import (
	"context"
	"fmt"
	"sort"
	"time"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

const (
	slowMovingMaxWindowDays = 365
	slowMovingMaxLimit      = 50
)

// slowMovingReporter narrows application/reports.GetSlowMovingProductsReportUseCase
// to the one method this use case needs (ISP).
type slowMovingReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.SlowMovingProductsReport, error)
}

// SlowMovingProduct is one in-stock SKU with zero sales in the queried
// window -- the shopper-safe subset of appreports.SlowMovingProductRow (no
// internal SKU IDs leave the storefront API).
type SlowMovingProduct struct {
	SKUCode     string
	ProductName string
	StockOnHand int64
}

type SlowMovingProductsResult struct {
	WindowDays int
	Products   []SlowMovingProduct
}

// ListSlowMovingProductsUseCase serves a storefront's "needs attention"
// merchandising block (e.g. the WooCommerce plugin's Elementor widget):
// in-stock SKUs nothing has sold of in the trailing `days`, most stock on
// hand first -- the products whose capital is tied up the most -- capped
// to `limit`. A thin, storefront-shaped view over the admin
// slow-moving-products report, not a separate computation.
type ListSlowMovingProductsUseCase struct {
	report slowMovingReporter
	now    func() time.Time
}

func NewListSlowMovingProductsUseCase(report slowMovingReporter, now func() time.Time) *ListSlowMovingProductsUseCase {
	return &ListSlowMovingProductsUseCase{report: report, now: now}
}

func (uc *ListSlowMovingProductsUseCase) Execute(ctx context.Context, days, limit int) (*SlowMovingProductsResult, error) {
	if days < 1 || days > slowMovingMaxWindowDays {
		return nil, fmt.Errorf("listSlowMovingProducts: %w", storefrontDomain.ErrInvalidSlowMovingWindow)
	}
	if limit < 1 || limit > slowMovingMaxLimit {
		return nil, fmt.Errorf("listSlowMovingProducts: %w", storefrontDomain.ErrInvalidSlowMovingLimit)
	}

	to := uc.now()
	report, err := uc.report.Execute(ctx, to.AddDate(0, 0, -days), to)
	if err != nil {
		return nil, fmt.Errorf("listSlowMovingProducts: %w", err)
	}

	rows := append([]appreports.SlowMovingProductRow(nil), report.Rows...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].StockOnHand > rows[j].StockOnHand })
	if len(rows) > limit {
		rows = rows[:limit]
	}

	products := make([]SlowMovingProduct, len(rows))
	for i, row := range rows {
		products[i] = SlowMovingProduct{SKUCode: row.SKUCode, ProductName: row.ProductName, StockOnHand: row.StockOnHand}
	}
	return &SlowMovingProductsResult{WindowDays: days, Products: products}, nil
}
