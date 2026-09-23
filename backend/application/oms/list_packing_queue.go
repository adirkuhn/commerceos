package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type packingQueueOrderLister interface {
	List(ctx context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error)
}

type openPickingSessionLister interface {
	ListOpenBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]omsDomain.PickingSession, error)
}

type activePackingSessionLister interface {
	ListActiveBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]omsDomain.PackingSession, error)
}

type ListPackingQueueResult struct {
	Entries []omsDomain.PackingQueueEntry
	Total   int64
}

// ListPackingQueueUseCase is the packing screens' order queue: every order
// still to be packed (see ListSalesOrdersFilter.AwaitingPacking), each with
// whoever is actively working it right now -- the UI shows those read-only
// to everyone else. Read-only.
type ListPackingQueueUseCase struct {
	db          *gorm.DB
	orderRepo   packingQueueOrderLister
	pickingRepo openPickingSessionLister
	packingRepo activePackingSessionLister
}

func NewListPackingQueueUseCase(db *gorm.DB, orderRepo packingQueueOrderLister, pickingRepo openPickingSessionLister, packingRepo activePackingSessionLister) *ListPackingQueueUseCase {
	return &ListPackingQueueUseCase{db: db, orderRepo: orderRepo, pickingRepo: pickingRepo, packingRepo: packingRepo}
}

func (uc *ListPackingQueueUseCase) Execute(ctx context.Context, page, perPage int) (*ListPackingQueueResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = defaultSalesOrdersPerPage
	}
	if perPage > maxSalesOrdersPerPage {
		perPage = maxSalesOrdersPerPage
	}

	result := &ListPackingQueueResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		orders, total, err := uc.orderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
			Limit: perPage, Offset: (page - 1) * perPage, AwaitingPacking: true,
		})
		if err != nil {
			return err
		}
		result.Total = total
		if len(orders) == 0 {
			return nil
		}

		ids := make([]string, len(orders))
		for i, o := range orders {
			ids[i] = o.ID
		}
		pickings, err := uc.pickingRepo.ListOpenBySalesOrders(ctx, ids)
		if err != nil {
			return err
		}
		packings, err := uc.packingRepo.ListActiveBySalesOrders(ctx, ids)
		if err != nil {
			return err
		}

		pickingByOrder := make(map[string]*omsDomain.PickingSession, len(pickings))
		for i := range pickings {
			pickingByOrder[pickings[i].SalesOrderID] = &pickings[i]
		}
		packingByOrder := make(map[string]*omsDomain.PackingSession, len(packings))
		for i := range packings {
			packingByOrder[packings[i].SalesOrderID] = &packings[i]
		}
		now := time.Now()
		result.Entries = make([]omsDomain.PackingQueueEntry, len(orders))
		for i, o := range orders {
			result.Entries[i] = omsDomain.NewPackingQueueEntry(o, pickingByOrder[o.ID], packingByOrder[o.ID], now)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listPackingQueue: %w", err)
	}
	return result, nil
}
