package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultSalesOrdersPerPage = 20
	maxSalesOrdersPerPage     = 200
)

type ListSalesOrdersInput struct {
	Page    int
	PerPage int
	// SortBy/SortDir empty means the repository's default (occurred_at,
	// desc -- newest first).
	SortBy  omsDomain.SalesOrderSortField
	SortDir omsDomain.SortDirection
	// NeedsStockReview/NeedsSKUReview nil means "don't filter" -- see
	// domain/oms.ListSalesOrdersFilter's doc.
	NeedsStockReview *bool
	NeedsSKUReview   *bool
	// Fulfilled nil means "don't filter" -- false powers the picking
	// scanner app's "unfulfilled orders" search.
	Fulfilled *bool
	// Packed nil means "don't filter" -- combined with Fulfilled=true,
	// false powers the packing screens' "orders ready to pack" search.
	Packed *bool
	// AwaitingPicking true narrows to the picking queue -- see
	// domain/oms.ListSalesOrdersFilter.AwaitingPicking's doc.
	AwaitingPicking bool
	// Status empty means "don't filter" -- see domain/oms.ListSalesOrdersFilter.Status's doc.
	Status []omsDomain.SalesOrderStatus
	// Source empty means "don't filter" -- see domain/oms.ListSalesOrdersFilter.Source's doc.
	Source []omsDomain.SalesOrderSource
	// CustomerID nil means "don't filter" -- see domain/oms.ListSalesOrdersFilter.CustomerID's doc.
	CustomerID *string
	// QuoteID nil means "don't filter" -- see domain/oms.ListSalesOrdersFilter.QuoteID.
	QuoteID *string
	// ShipDue true narrows to orders that must ship today or are late --
	// ListSalesOrdersFilter.ShipDueOnOrBefore with today's date in the
	// tenant's reporting timezone. false means "don't filter".
	ShipDue bool
}

// tenantSettingsReader is the one settings call listing needs: the
// tenant's reporting timezone, which decides what "today" is for ShipDue.
type tenantSettingsReader interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type ListSalesOrdersResult struct {
	SalesOrders []omsDomain.SalesOrder
	Total       int64
}

// ListSalesOrdersUseCase is the read side for staff viewing what came in
// from POS (or, later, any other channel) -- ReceivePOSSalesOrderUseCase is
// the only write path today, this never creates or mutates anything.
type ListSalesOrdersUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	settings       tenantSettingsReader
	// Now is the injectable clock ShipDue's "today" is computed from --
	// defaults to time.Now, overridden in tests.
	Now func() time.Time
}

func NewListSalesOrdersUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, settings tenantSettingsReader) *ListSalesOrdersUseCase {
	return &ListSalesOrdersUseCase{db: db, salesOrderRepo: salesOrderRepo, settings: settings, Now: time.Now}
}

func (uc *ListSalesOrdersUseCase) Execute(ctx context.Context, in ListSalesOrdersInput) (*ListSalesOrdersResult, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	perPage := in.PerPage
	if perPage <= 0 {
		perPage = defaultSalesOrdersPerPage
	}
	if perPage > maxSalesOrdersPerPage {
		perPage = maxSalesOrdersPerPage
	}

	result := &ListSalesOrdersResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		filter := omsDomain.ListSalesOrdersFilter{
			Limit: perPage, Offset: (page - 1) * perPage, SortBy: in.SortBy, SortDir: in.SortDir,
			NeedsStockReview: in.NeedsStockReview, NeedsSKUReview: in.NeedsSKUReview, Fulfilled: in.Fulfilled, Packed: in.Packed, AwaitingPicking: in.AwaitingPicking,
			Status: in.Status, Source: in.Source, CustomerID: in.CustomerID, QuoteID: in.QuoteID,
		}
		if in.ShipDue {
			settings, err := uc.settings.Get(ctx)
			if err != nil {
				return err
			}
			today := settings.Today(uc.Now())
			filter.ShipDueOnOrBefore = &today
		}
		orders, total, err := uc.salesOrderRepo.List(ctx, filter)
		if err != nil {
			return err
		}
		result.SalesOrders = orders
		result.Total = total
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listSalesOrders: %w", err)
	}
	return result, nil
}
