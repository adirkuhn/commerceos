package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── SalesOrderRepo ───────────────────────────────────────────────────────────

type SalesOrderRepo struct{ db *gorm.DB }

func NewSalesOrderRepo(db *gorm.DB) *SalesOrderRepo { return &SalesOrderRepo{db: db} }

func (r *SalesOrderRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SalesOrderRepo) Create(ctx context.Context, o *omsDomain.SalesOrder) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := salesOrderModelFromDomain(o)
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("salesOrderRepo.Create: %w", err)
	}
	o.ID = m.ID
	return nil
}

// TrySetStockDecremented atomically flips StockDecremented from `from` to
// `to`, conditioned on the row's current value still being `from` -- the
// DB-level guard against the double-decrement race two concurrent callers
// of UpdateSalesOrderUseCase could otherwise hit (both reading
// StockDecremented=false via FindByID before either's write lands). Returns
// false (not an error) when the row's value had already moved on -- the
// caller's job is done already, by whoever won the race.
func (r *SalesOrderRepo) TrySetStockDecremented(ctx context.Context, id string, from, to bool) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	res := tx.Model(&salesOrderModel{}).
		Where("id = ? AND stock_decremented = ?", id, from).
		Updates(map[string]any{"stock_decremented": to, "updated_at": time.Now()})
	if res.Error != nil {
		return false, fmt.Errorf("salesOrderRepo.TrySetStockDecremented: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *SalesOrderRepo) FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, omsDomain.ErrSalesOrderNotFound
		}
		return nil, fmt.Errorf("salesOrderRepo.FindByID: %w", err)
	}
	return toSalesOrderDomain(m), nil
}

func (r *SalesOrderRepo) FindByNumber(ctx context.Context, number string) (*omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderModel
	err = tx.Where("number = ?", number).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("salesOrderRepo.FindByNumber: %w", err)
	}
	return toSalesOrderDomain(m), nil
}

// salesOrderOrderClause maps a filter's SortBy/SortDir to a safe ORDER BY
// clause via a closed switch over omsDomain.SalesOrderSortField -- never
// interpolates either value directly, so an invalid/unknown field or
// direction can't reach raw SQL. Unset/unknown SortBy defaults to
// occurred_at; unset/unknown SortDir defaults to desc (newest first).
func salesOrderOrderClause(filter omsDomain.ListSalesOrdersFilter) string {
	dir := "desc"
	if filter.SortDir == omsDomain.SortAsc {
		dir = "asc"
	}
	switch filter.SortBy {
	case omsDomain.SalesOrderSortCustomer:
		return "billing_last_name " + dir + ", billing_first_name " + dir
	case omsDomain.SalesOrderSortSource:
		return salesOrderOriginSQL("sales_orders") + " " + dir
	case omsDomain.SalesOrderSortStatus:
		return "status " + dir
	case omsDomain.SalesOrderSortTotal:
		return "total_minor_units " + dir
	default:
		return "occurred_at " + dir
	}
}

func salesOrderStatusStrings(statuses []omsDomain.SalesOrderStatus) []string {
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = string(s)
	}
	return out
}

// salesOrdersMatching narrows sales_orders to the rows a
// ListSalesOrdersFilter selects -- the one place those rules live, shared
// by List and Count so a queue's count can never drift from its list.
// Limit/Offset/Sort are the caller's business.
func salesOrdersMatching(tx *gorm.DB, filter omsDomain.ListSalesOrdersFilter) *gorm.DB {
	q := tx.Model(&salesOrderModel{})
	if filter.NeedsStockReview != nil {
		q = q.Where("needs_stock_review = ?", *filter.NeedsStockReview)
	}
	if filter.NeedsSKUReview != nil {
		q = q.Where("needs_sku_review = ?", *filter.NeedsSKUReview)
	}
	if len(filter.Source) > 0 {
		sources := make([]string, len(filter.Source))
		for i, s := range filter.Source {
			sources[i] = string(s)
		}
		q = q.Where(salesOrderOriginSQL("sales_orders")+" IN ?", sources)
	}
	if filter.Fulfilled != nil {
		q = q.Where("fulfilled = ?", *filter.Fulfilled)
	}
	if filter.Packed != nil {
		q = q.Where("packed = ?", *filter.Packed)
	}
	if filter.AwaitingPicking {
		q = q.Where("fulfilled = false").
			Where("status IN ?", salesOrderStatusStrings(omsDomain.PickableStatuses())).
			Where(notPOSSaleSQL("sales_orders"))
	}
	if filter.AwaitingPacking {
		q = q.Where("packed = false").
			Where("status IN ?", salesOrderStatusStrings(omsDomain.PickableStatuses())).
			Where(notPOSSaleSQL("sales_orders"))
	}
	if len(filter.Status) > 0 {
		q = q.Where("status IN ?", salesOrderStatusStrings(filter.Status))
	}
	if filter.CustomerID != nil {
		q = q.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.QuoteID != nil {
		q = q.Where("quote_id = ?", *filter.QuoteID)
	}
	if filter.SalesChannelID != nil {
		q = q.Where("sales_channel_id = ?", *filter.SalesChannelID)
	}
	if filter.ShipDueOnOrBefore != nil {
		q = q.Where("ship_by_date IS NOT NULL").
			Where("ship_by_date <= ?::date", filter.ShipDueOnOrBefore.Format(time.DateOnly)).
			Where("status IN ?", salesOrderStatusStrings(omsDomain.PickableStatuses())).
			Where(notPOSSaleSQL("sales_orders"))
	}
	return q
}

var _ omsDomain.SalesOrderCounter = (*SalesOrderRepo)(nil)

// Count is List's total without loading any rows -- see
// omsDomain.SalesOrderCounter.
func (r *SalesOrderRepo) Count(ctx context.Context, filter omsDomain.ListSalesOrdersFilter) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	if err := salesOrdersMatching(tx, filter).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("salesOrderRepo.Count: %w", err)
	}
	return total, nil
}

func (r *SalesOrderRepo) List(ctx context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := salesOrdersMatching(tx, filter)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("salesOrderRepo.List: count: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	var rows []salesOrderModel
	if err := q.Order(salesOrderOrderClause(filter)).Limit(limit).Offset(filter.Offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("salesOrderRepo.List: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, total, nil
}

var _ omsDomain.MissingPaymentInfoLister = (*SalesOrderRepo)(nil)

func (r *SalesOrderRepo) ListMissingPaymentInfo(ctx context.Context, connectionID string, after *omsDomain.MissingPaymentInfoCursor, limit int) ([]omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Model(&salesOrderModel{}).
		Where(fromConnectionSQL("sales_orders"), connectionID).
		Where("transaction_id = '' AND date_paid IS NULL").
		Where("status IN ?", salesOrderStatusStrings(omsDomain.PaymentRecordExpectedStatuses()))
	if after != nil {
		q = q.Where("(occurred_at, id) < (?, ?)", after.OccurredAt, after.ID)
	}
	var rows []salesOrderModel
	if err := q.Order("occurred_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ListMissingPaymentInfo: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, nil
}

func (r *SalesOrderRepo) ListByConnection(ctx context.Context, connectionID string) ([]omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesOrderModel
	if err := tx.Where(fromConnectionSQL("sales_orders"), connectionID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ListByConnection: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, nil
}

func (r *SalesOrderRepo) Search(ctx context.Context, query string, limit int) ([]omsDomain.SalesOrder, error) {
	if query == "" {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}
	pattern := "%" + query + "%"
	var rows []salesOrderModel
	err = tx.Where(
		"unaccent(number) ILIKE unaccent(?) OR "+
			"similarity(immutable_unaccent(number), immutable_unaccent(?)) > ? OR "+
			"unaccent(billing_first_name || ' ' || billing_last_name) ILIKE unaccent(?) OR "+
			"similarity(immutable_unaccent(billing_first_name || ' ' || billing_last_name), immutable_unaccent(?)) > ? OR "+
			"billing_email ILIKE ?",
		pattern, query, fuzzySearchSimilarityThreshold,
		pattern, query, fuzzySearchSimilarityThreshold,
		pattern,
	).Clauses(clause.OrderBy{
		Expression: clause.Expr{
			SQL: "GREATEST(" +
				"similarity(immutable_unaccent(number), immutable_unaccent(?)), " +
				"similarity(immutable_unaccent(billing_first_name || ' ' || billing_last_name), immutable_unaccent(?))" +
				") DESC, occurred_at desc",
			Vars: []interface{}{query, query},
		},
	}).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.Search: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, nil
}

func (r *SalesOrderRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&salesOrderModel{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("salesOrderRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return omsDomain.ErrSalesOrderNotFound
	}
	return nil
}

func (r *SalesOrderRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&salesOrderModel{})
	if res.Error != nil {
		return fmt.Errorf("salesOrderRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return omsDomain.ErrSalesOrderNotFound
	}
	return nil
}
