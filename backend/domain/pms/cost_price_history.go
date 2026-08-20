package pms

import "time"

// SKUCostPriceHistory is one daily snapshot of a SKU's computed cost price
// and recommended price, written by application/pms's
// RecalculateCostPricesUseCase (the daily cron) -- the margin-trend-over-time
// data source. One row per SKU per calendar day the cron actually computed a
// value for that SKU (a SKU with zero on-hand batches that day is skipped,
// not written as a zero/nil row).
type SKUCostPriceHistory struct {
	ID                         string
	TenantID                   string
	SKUID                      string
	CostPriceMinorUnits        int64
	RecommendedPriceMinorUnits int64
	// OccurredOn is a calendar date (time truncated to midnight UTC), not a
	// timestamp -- (tenant_id, sku_id, occurred_on) is unique, which is
	// what makes a same-day cron rerun idempotent (see
	// SKUCostPriceHistoryRepository.Upsert).
	OccurredOn time.Time
	CreatedAt  time.Time
}
