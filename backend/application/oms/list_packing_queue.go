package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type queueOrderLister interface {
	List(ctx context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error)
}

type openPickingSessionLister interface {
	ListOpenBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]omsDomain.PickingSession, error)
}

type activePackingSessionLister interface {
	ListActiveBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]omsDomain.PackingSession, error)
}

type packSessionScanChecker interface {
	SessionIDsWithScans(ctx context.Context, sessionIDs []string) ([]string, error)
}

// ListQueueResult is one page of the picking or packing queue.
type ListQueueResult struct {
	Entries []omsDomain.QueueEntry
	Total   int64
}

// orderQueue lists one page of orders matching a queue filter, each with
// who is working it (see domain/oms.NewQueueEntry). Shared by the picking
// and packing queues -- they differ only in which orders they list.
type orderQueue struct {
	db          *gorm.DB
	orderRepo   queueOrderLister
	pickingRepo openPickingSessionLister
	packingRepo activePackingSessionLister
	packScans   packSessionScanChecker
}

func (q orderQueue) list(ctx context.Context, filter omsDomain.ListSalesOrdersFilter, page, perPage int) (*ListQueueResult, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = defaultSalesOrdersPerPage
	}
	if perPage > maxSalesOrdersPerPage {
		perPage = maxSalesOrdersPerPage
	}
	filter.Limit, filter.Offset = perPage, (page-1)*perPage

	result := &ListQueueResult{}
	err := tenant.WithTenant(ctx, q.db, func(ctx context.Context) error {
		orders, total, err := q.orderRepo.List(ctx, filter)
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
		pickings, err := q.pickingRepo.ListOpenBySalesOrders(ctx, ids)
		if err != nil {
			return err
		}
		packings, err := q.packingRepo.ListActiveBySalesOrders(ctx, ids)
		if err != nil {
			return err
		}
		packingIDs := make([]string, len(packings))
		for i, p := range packings {
			packingIDs[i] = p.ID
		}
		withScans, err := q.packScans.SessionIDsWithScans(ctx, packingIDs)
		if err != nil {
			return err
		}
		hasScans := make(map[string]bool, len(withScans))
		for _, id := range withScans {
			hasScans[id] = true
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
		result.Entries = make([]omsDomain.QueueEntry, len(orders))
		for i, o := range orders {
			packing := packingByOrder[o.ID]
			result.Entries[i] = omsDomain.NewQueueEntry(o, pickingByOrder[o.ID], packing, packing != nil && hasScans[packing.ID], now)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListPackingQueueUseCase is the packing screens' order queue: every order
// still to be packed (see ListSalesOrdersFilter.AwaitingPacking), each with
// whoever is actively working it -- the UI shows those read-only to
// everyone else. Read-only.
type ListPackingQueueUseCase struct{ queue orderQueue }

func NewListPackingQueueUseCase(db *gorm.DB, orderRepo queueOrderLister, pickingRepo openPickingSessionLister, packingRepo activePackingSessionLister, packScans packSessionScanChecker) *ListPackingQueueUseCase {
	return &ListPackingQueueUseCase{queue: orderQueue{db: db, orderRepo: orderRepo, pickingRepo: pickingRepo, packingRepo: packingRepo, packScans: packScans}}
}

func (uc *ListPackingQueueUseCase) Execute(ctx context.Context, page, perPage int) (*ListQueueResult, error) {
	result, err := uc.queue.list(ctx, omsDomain.ListSalesOrdersFilter{AwaitingPacking: true}, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("listPackingQueue: %w", err)
	}
	return result, nil
}

// ListPickingQueueUseCase is the picking screens' order queue: every order
// still to be picked (see ListSalesOrdersFilter.AwaitingPicking), each with
// whoever is working it and whether packing has it (InPacking) -- the UI
// shows those as a read-only item list. Read-only.
type ListPickingQueueUseCase struct{ queue orderQueue }

func NewListPickingQueueUseCase(db *gorm.DB, orderRepo queueOrderLister, pickingRepo openPickingSessionLister, packingRepo activePackingSessionLister, packScans packSessionScanChecker) *ListPickingQueueUseCase {
	return &ListPickingQueueUseCase{queue: orderQueue{db: db, orderRepo: orderRepo, pickingRepo: pickingRepo, packingRepo: packingRepo, packScans: packScans}}
}

func (uc *ListPickingQueueUseCase) Execute(ctx context.Context, page, perPage int) (*ListQueueResult, error) {
	result, err := uc.queue.list(ctx, omsDomain.ListSalesOrdersFilter{AwaitingPicking: true}, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("listPickingQueue: %w", err)
	}
	return result, nil
}
