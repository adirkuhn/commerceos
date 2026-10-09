package oms

import (
	"context"
	"time"
)

// SalesOrderSortField is the allow-list of columns List can sort by --
// kept as a closed type (rather than a raw string) so an invalid/unknown
// value can never reach the repository's ORDER BY clause.
type SalesOrderSortField string

const (
	SalesOrderSortOccurredAt SalesOrderSortField = "occurred_at"
	SalesOrderSortCustomer   SalesOrderSortField = "customer"
	SalesOrderSortSource     SalesOrderSortField = "source"
	SalesOrderSortStatus     SalesOrderSortField = "status"
	SalesOrderSortTotal      SalesOrderSortField = "total"
)

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type ListSalesOrdersFilter struct {
	Limit  int
	Offset int
	// SortBy/SortDir empty means the repository's default (occurred_at,
	// desc -- newest first).
	SortBy  SalesOrderSortField
	SortDir SortDirection
	// NeedsStockReview/NeedsSKUReview nil means "don't filter on this" --
	// set to a non-nil true to power the "Stock Issues"/"Unmatched SKUs"
	// admin pages, which list only flagged orders.
	NeedsStockReview *bool
	NeedsSKUReview   *bool
	// Source empty means "don't filter on this" -- non-empty narrows to
	// those sources only (OR'd together, same convention as Status). Used
	// by cmd/repair-stock-review-flags to scan only WooCommerce-sourced
	// orders, and by the Sales Orders page's channel filter.
	Source []SalesOrderSource
	// Fulfilled nil means "don't filter on this" -- false powers the
	// picking scanner app's "unfulfilled orders" search; also usable from
	// the OMS Orders page's own filter.
	Fulfilled *bool
	// Packed nil means "don't filter on this" -- combined with fulfilled=
	// true, false powers the packing screens' "orders ready to pack" search
	// (see SalesOrder.Packed's doc).
	Packed *bool
	// AwaitingPacking true narrows to the packing queue -- the SQL form of
	// SalesOrder.CanStartPacking: not packed, not a POS sale, and still in a
	// pickable status (see PickableStatuses), picked or not. Orders being
	// picked right now are included; the queue shows them locked (see
	// QueueEntry). false means "don't filter on this".
	AwaitingPacking bool
	// AwaitingPicking true narrows to the picking queue -- the SQL form of
	// SalesOrder.CanMarkPicked: not yet picked (Fulfilled false), still in a
	// pickable status, and not a POS sale. false means "don't filter on
	// this".
	AwaitingPicking bool
	// Status empty means "don't filter on this". Non-empty narrows to those
	// statuses only -- powers the picking screens' exclusion of
	// completed/cancelled/refunded/failed orders (a terminal-status order
	// must never be offered for picking, regardless of its Fulfilled flag).
	Status []SalesOrderStatus
	// CustomerID nil means "don't filter on this" -- set to narrow to one
	// customer's orders, e.g. the Customer detail page's order history and
	// lifetime-spend figures. Guest orders (SalesOrder.CustomerID nil) never
	// match a non-nil filter value.
	CustomerID *string
	// SalesChannelID nil means "don't filter on this" -- set to narrow to
	// one internal storefront channel, e.g. a logged-in customer's account
	// order history (application/storefront.ListAccountOrdersUseCase),
	// same channel-isolation rule LookupOrderUseCase's own doc comment
	// enforces for guest lookup: a B2C channel must never surface a B2B
	// order, or vice versa.
	SalesChannelID *string
}

type SalesOrderRepository interface {
	Create(ctx context.Context, o *SalesOrder) error
	FindByID(ctx context.Context, id string) (*SalesOrder, error)
	// FindByExternalID looks up an already-received sale by its source's
	// own reference -- the idempotency check ReceivePOSSalesOrderUseCase
	// runs before creating anything. Nil (not an error) when none exists.
	FindByExternalID(ctx context.Context, source SalesOrderSource, externalOrderID string) (*SalesOrder, error)
	// FindByNumber looks up a sale by its human-facing Number -- a payment
	// webhook's own reference back to the order it paid for (Stripe's
	// client_reference_id, Revolut's merchant_order_ext_ref) is always this
	// Number, never the internal ID. Nil (not an error) when none exists.
	FindByNumber(ctx context.Context, number string) (*SalesOrder, error)
	// List returns every sales order for the current tenant (RLS-scoped),
	// newest first, paginated.
	List(ctx context.Context, filter ListSalesOrdersFilter) ([]SalesOrder, int64, error)
	// Search returns up to limit sales orders whose order number or billing
	// name matches query -- number/name matching is accent-insensitive +
	// typo-tolerant (pg_trgm), same convention as
	// pms.ProductRepository.List's Search filter. Ranked best match first.
	// Powers global search (Cmd+K). Empty query returns nil, not an error.
	Search(ctx context.Context, query string, limit int) ([]SalesOrder, error)
	// ListByConnection returns every order stamped with this connection's
	// ID (see SalesOrder.ConnectionID) -- used by "delete all imported
	// products" to find that connection's own orders. Unpaginated, same
	// convention as channels.ProductChannelListingRepository.ListByConnection.
	ListByConnection(ctx context.Context, connectionID string) ([]SalesOrder, error)
	// Update patches header fields (status, customer, billing/shipping,
	// note, NeedsStockReview, NeedsSKUReview) -- never line items, which
	// are immutable once created (see UpdateSalesOrderUseCase's doc), and
	// never StockDecremented, which must only ever change through
	// TrySetStockDecremented's conditional write.
	Update(ctx context.Context, id string, fields map[string]any) error
	// TrySetStockDecremented atomically flips StockDecremented from
	// `from` to `to`, only if the row's current value is still `from` --
	// see the implementation's doc for why this can't be a plain field in
	// Update's fields map (the double-decrement race it closes).
	TrySetStockDecremented(ctx context.Context, id string, from, to bool) (bool, error)
	Delete(ctx context.Context, id string) error
}

// ChannelRevenue is one row of DashboardSalesReportRepository.RevenueByChannelSince
// -- total revenue for one source (channel) within the queried window,
// already narrowed to a single currency by the caller.
type ChannelRevenue struct {
	Source          SalesOrderSource
	TotalMinorUnits int64
}

// ProductSales is one row of DashboardSalesReportRepository.TopProductsSince
// -- SKUID/SKUCode/ProductName are read straight off the line item's own
// Name/SKUCode snapshot (see SalesOrderLineItem's doc), not joined from
// pms, so a since-renamed or deleted product still reports correctly.
// SKUID is nil for an adhoc/unmatched line item, same convention as
// SalesOrderLineItem.SKUID.
type ProductSales struct {
	SKUID             *string
	SKUCode           string
	ProductName       string
	QuantitySold      int64
	RevenueMinorUnits int64
}

// DashboardSalesReportRepository backs the dashboard's revenue/channel/
// top-products figures -- read-only aggregate queries over sales_orders
// and sales_order_line_items, scoped to orders whose status counts as
// revenue (see SalesOrderStatus.CountsAsRevenue / RevenueCountingStatuses;
// a pending/cancelled/refunded/failed order was never real revenue) and to
// a single currency (a tenant reporting in more than one currency has
// orders outside that currency excluded from these figures entirely --
// see application/dashboard.GetDashboardStatsUseCase's doc).
type DashboardSalesReportRepository interface {
	RevenueSince(ctx context.Context, since time.Time, currencyCode string) (int64, error)
	RevenueByChannelSince(ctx context.Context, since time.Time, currencyCode string) ([]ChannelRevenue, error)
	// TopProductsSince ranks by quantity sold (currency-agnostic), highest
	// first, limited to `limit` rows.
	TopProductsSince(ctx context.Context, since time.Time, currencyCode string, limit int) ([]ProductSales, error)
}

type SalesOrderLineItemRepository interface {
	Create(ctx context.Context, li *SalesOrderLineItem) error
	FindByID(ctx context.Context, id string) (*SalesOrderLineItem, error)
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]SalesOrderLineItem, error)
	DeleteBySalesOrder(ctx context.Context, salesOrderID string) error
	// Update patches a line item's mutable fields -- today only SKUID
	// (LinkLineItemSKUUseCase) and StockStatus (RetryStockDecrementUseCase,
	// ImportWooOrdersUseCase, UpdateSalesOrderUseCase). Everything else
	// about a line item is immutable once created (see SalesOrderRepository
	// .Update's doc).
	Update(ctx context.Context, id string, fields map[string]any) error
}

// SalesOrderLineItemStockMovement links a line item to the stock movement
// it caused -- see the migration's doc comment for why this is a dedicated
// link table rather than a column on stock_movements.
type SalesOrderLineItemStockMovement struct {
	ID                   string
	TenantID             string
	SalesOrderLineItemID string
	StockMovementID      string
	// BatchID and QuantityDelta mirror the linked StockMovement's own
	// fields at the time it was created -- denormalized so a restock can
	// reverse exactly what was actually decremented (which real batch, how
	// much) without a second lookup into stock_movements. nil BatchID means
	// the general/un-lotted ledger, same convention as inventory.Batch.
	BatchID       *string
	QuantityDelta int64
	CreatedAt     time.Time
}

type SalesOrderLineItemStockMovementRepository interface {
	Create(ctx context.Context, l *SalesOrderLineItemStockMovement) error
	// ListByLineItem returns every movement linked to lineItemID, in the
	// order they were created -- used to reverse a decrement batch-by-batch
	// when an order restocks (see application/oms's restock helpers).
	ListByLineItem(ctx context.Context, lineItemID string) ([]SalesOrderLineItemStockMovement, error)
	// ExistsForLineItem reports whether lineItemID already has a recorded
	// stock movement linked to it -- used by
	// cmd/repair-stock-review-flags to distinguish a line item whose
	// decrement genuinely never ran (the bug this repairs) from one that
	// already succeeded.
	ExistsForLineItem(ctx context.Context, lineItemID string) (bool, error)
}

type PickingSessionRepository interface {
	Create(ctx context.Context, s *PickingSession) error
	FindByID(ctx context.Context, id string) (*PickingSession, error)
	// FindActiveBySalesOrder returns the order's current in_progress or
	// paused session, if any -- StartOrResumePickingUseCase reuses it
	// instead of creating a duplicate. Nil (not an error) when none exists.
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*PickingSession, error)
	Update(ctx context.Context, s *PickingSession) error
	// AcquireLock claims (or refreshes) the session's lock for userID,
	// unconditionally -- callers must check PickingSession.LockedBy/
	// LockedAt themselves first (see StartOrResumePickingUseCase/
	// TouchPickingSessionUseCase) before calling this. A dedicated
	// map-based update, not Update, because Update's struct-based
	// .Updates() silently skips nil-pointer (zero-value) fields -- it
	// could never clear LockedBy back to NULL on release. Sets LockedSince
	// to at only when the holder changes, so heartbeats keep it.
	AcquireLock(ctx context.Context, id, userID string, at time.Time) error
	// ReleaseLock clears the session's lock unconditionally -- see
	// AcquireLock's doc for why this can't just be Update.
	ReleaseLock(ctx context.Context, id string) error
	// ListOpenBySalesOrders returns the in_progress/paused sessions of the
	// given orders -- the packing queue's lock lookup for one page.
	ListOpenBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]PickingSession, error)
}

type PickScanRepository interface {
	Create(ctx context.Context, s *PickScan) error
	// ListBySession returns every scan for the session in ScannedAt order
	// (oldest first) -- CompletePickingUseCase's replay depends on this
	// ordering.
	ListBySession(ctx context.Context, sessionID string) ([]PickScan, error)
	// MarkReplayed records that CompletePickingUseCase's replay applied
	// this row for real -- see PickScan.ReplayedAt's doc.
	MarkReplayed(ctx context.Context, id string, replayedAt time.Time) error
	// Delete removes a staged row before it's ever been replayed -- see
	// DeletePickScanUseCase, mirrors procurement.BookingScanRepository.Delete.
	Delete(ctx context.Context, id string) error
}

// PackingSessionRepository mirrors PickingSessionRepository exactly -- see
// that interface's doc comments, identical here.
type PackingSessionRepository interface {
	Create(ctx context.Context, s *PackingSession) error
	FindByID(ctx context.Context, id string) (*PackingSession, error)
	// FindActiveBySalesOrder returns the order's current in_progress
	// session, if any. Nil (not an error) when none exists.
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*PackingSession, error)
	Update(ctx context.Context, s *PackingSession) error
	AcquireLock(ctx context.Context, id, userID string, at time.Time) error
	ReleaseLock(ctx context.Context, id string) error
	// ListActiveBySalesOrders returns the in_progress sessions of the given
	// orders -- the packing queue's lock lookup for one page.
	ListActiveBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]PackingSession, error)
}

// PackScanRepository is PickScanRepository without the replay bookkeeping
// (see PackScan's doc comment for why none is needed).
type PackScanRepository interface {
	Create(ctx context.Context, s *PackScan) error
	// ListBySession returns every scan for the session in ScannedAt order.
	ListBySession(ctx context.Context, sessionID string) ([]PackScan, error)
	// Delete removes a scan -- see DeletePackScanUseCase.
	Delete(ctx context.Context, id string) error
	// MarkReplayed records that CompletePackingUseCase applied this row's
	// order change -- see PackScan.ReplayedAt.
	MarkReplayed(ctx context.Context, id string, replayedAt time.Time) error
	// SessionIDsWithScans returns which of the given sessions have at least
	// one scan -- see PackingSession.BlocksPicking.
	SessionIDsWithScans(ctx context.Context, sessionIDs []string) ([]string, error)
}

// MissingPaymentInfoCursor resumes a MissingPaymentInfoLister listing after
// the last order already seen (newest first: OccurredAt desc, ID desc).
type MissingPaymentInfoCursor struct {
	OccurredAt time.Time
	ID         string
}

// MissingPaymentInfoLister finds one channel connection's orders that went
// through a payment (SalesOrderStatus.ExpectsPaymentRecord) yet carry no
// transaction id and no paid date -- e.g. imported while pending, before
// the import re-synced payment info. Kept apart from SalesOrderRepository
// so only the payment backfill depends on it.
type MissingPaymentInfoLister interface {
	ListMissingPaymentInfo(ctx context.Context, connectionID string, after *MissingPaymentInfoCursor, limit int) ([]SalesOrder, error)
}
