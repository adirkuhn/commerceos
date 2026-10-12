package dashboard

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// expiringBatchesWindowDays is the "expiring soon" window the dashboard has
// always used (the frontend called the expiring-batches endpoint with
// days=7).
const expiringBatchesWindowDays = 7

// TodayQueue answers "what does the shop need to do right now" -- one count
// per work queue, each using the exact rule that queue's own page lists
// with:
//   - ToPick: the picking queue (ListSalesOrdersFilter.AwaitingPicking)
//   - ToPack: the packing queue (ListSalesOrdersFilter.AwaitingPacking)
//   - StockIssues: orders flagged NeedsStockReview (the Stock issues page)
//   - UnmatchedSKUs: orders flagged NeedsSKUReview (the Unmatched SKUs
//     page) -- it counts orders, not line items
//   - ExpiringBatches: BatchRepository.ListExpiringSoon(7) -- in-stock
//     batches expiring within 7 days, already-past-expiry ones included
//   - ExpiredBatches: BatchRepository.ListPendingReview -- expired batches
//     nobody has disposed of/dismissed yet
type TodayQueue struct {
	ToPick          int64
	ToPack          int64
	StockIssues     int64
	UnmatchedSKUs   int64
	ExpiringBatches int64
	ExpiredBatches  int64
}

// GetTodayQueueUseCase backs the admin's "Today" dashboard. Read-only; every
// count runs in one tenant transaction so they're a consistent snapshot.
type GetTodayQueueUseCase struct {
	db      *gorm.DB
	orders  omsDomain.SalesOrderCounter
	batches invDomain.BatchQueueCounter
}

func NewGetTodayQueueUseCase(db *gorm.DB, orders omsDomain.SalesOrderCounter, batches invDomain.BatchQueueCounter) *GetTodayQueueUseCase {
	return &GetTodayQueueUseCase{db: db, orders: orders, batches: batches}
}

func (uc *GetTodayQueueUseCase) Execute(ctx context.Context) (*TodayQueue, error) {
	flagged := true
	orderQueues := []struct {
		filter omsDomain.ListSalesOrdersFilter
		dst    func(*TodayQueue) *int64
	}{
		{omsDomain.ListSalesOrdersFilter{AwaitingPicking: true}, func(q *TodayQueue) *int64 { return &q.ToPick }},
		{omsDomain.ListSalesOrdersFilter{AwaitingPacking: true}, func(q *TodayQueue) *int64 { return &q.ToPack }},
		{omsDomain.ListSalesOrdersFilter{NeedsStockReview: &flagged}, func(q *TodayQueue) *int64 { return &q.StockIssues }},
		{omsDomain.ListSalesOrdersFilter{NeedsSKUReview: &flagged}, func(q *TodayQueue) *int64 { return &q.UnmatchedSKUs }},
	}

	var result TodayQueue
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, oq := range orderQueues {
			n, err := uc.orders.Count(ctx, oq.filter)
			if err != nil {
				return err
			}
			*oq.dst(&result) = n
		}
		var err error
		if result.ExpiringBatches, err = uc.batches.CountExpiringSoon(ctx, expiringBatchesWindowDays); err != nil {
			return err
		}
		result.ExpiredBatches, err = uc.batches.CountPendingReview(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getTodayQueue: %w", err)
	}
	return &result, nil
}
