package channels

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// repushWooOrderStatusesLimit bounds one sweep to the newest orders --
// an outage strands recent orders, not years-old ones.
const repushWooOrderStatusesLimit = 500

// salesOrderLister is SalesOrderRepository.List narrowed to point of use.
type salesOrderLister interface {
	List(ctx context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error)
}

// salesOrderStatusPusher pushes an order's current status to its channel
// (PushSalesOrderStatusToChannelsUseCase).
type salesOrderStatusPusher interface {
	Execute(ctx context.Context, salesOrderID string) error
}

// RepushWooOrderStatusesUseCase is the self-heal sweep for status pushes
// that never reached Woo. The push consumer gives up after a few failed
// deliveries (dead-letter), so a store outage longer than its backoff
// (2026-10-31..11-03: three days of 403) left Woo on processing for orders
// we had shipped. This sweep compares Woo's current status with ours and
// re-pushes where Woo is behind -- exactly the cases the order import
// refuses to take from Woo (see wooStatusNeedsRepush), so the two never
// fight. Failures are logged, not fatal: the next run retries.
//
//events:delegated PushSalesOrderStatusToChannelsUseCase -- records channel.order_status_pushed per successful push
type RepushWooOrderStatusesUseCase struct {
	db      *gorm.DB
	orders  salesOrderLister
	clients wooClientResolver
	pusher  salesOrderStatusPusher
}

func NewRepushWooOrderStatusesUseCase(db *gorm.DB, orders salesOrderLister, clients wooClientResolver, pusher salesOrderStatusPusher) *RepushWooOrderStatusesUseCase {
	return &RepushWooOrderStatusesUseCase{db: db, orders: orders, clients: clients, pusher: pusher}
}

func (uc *RepushWooOrderStatusesUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var candidates []omsDomain.SalesOrder
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		candidates, _, err = uc.orders.List(ctx, omsDomain.ListSalesOrdersFilter{
			Source: []omsDomain.SalesOrderSource{omsDomain.SalesOrderSourceWooCommerce},
			// The statuses a channel can fall behind on and we can push
			// (PartiallyRefunded has no Woo equivalent to push).
			Status: []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusShipped, omsDomain.SalesOrderStatusCompleted},
			Limit:  repushWooOrderStatusesLimit,
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("repushWooOrderStatuses: %w", err)
	}

	for connectionID, orders := range groupByConnection(candidates) {
		uc.repushConnection(tenant.WithTenantID(adminCtx, orders[0].TenantID), connectionID, orders)
	}
	return nil
}

func (uc *RepushWooOrderStatusesUseCase) repushConnection(ctx context.Context, connectionID string, orders []omsDomain.SalesOrder) {
	client, err := uc.clients.ClientFor(ctx, connectionID)
	if err != nil {
		log.Printf("repushWooOrderStatuses: connection %s: %v", connectionID, err)
		return
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ExternalOrderID
	}
	wooStatuses, err := client.ListOrderStatuses(ctx, ids)
	if err != nil {
		log.Printf("repushWooOrderStatuses: connection %s: read Woo statuses: %v", connectionID, err)
		return
	}
	for _, o := range orders {
		wooStatus, onWoo := wooStatuses[o.ExternalOrderID]
		if !onWoo || !wooStatusNeedsRepush(o.Status, wooStatus) {
			continue
		}
		if err := uc.pusher.Execute(ctx, o.ID); err != nil {
			log.Printf("repushWooOrderStatuses: order %s (woo %s): %v", o.ID, o.ExternalOrderID, err)
		}
	}
}

// wooStatusNeedsRepush: Woo differs from what we'd push, and our import
// would refuse Woo's status (Woo is stale, not ahead). When Woo is ahead
// (e.g. cancelled there) the import brings it in instead.
func wooStatusNeedsRepush(local omsDomain.SalesOrderStatus, wooStatus string) bool {
	want, pushable := wooStatusBySalesOrderStatus[local]
	if !pushable || want == wooStatus {
		return false
	}
	return !local.AcceptsChannelStatus(omsDomain.SalesOrderStatus(wooStatus))
}

func groupByConnection(orders []omsDomain.SalesOrder) map[string][]omsDomain.SalesOrder {
	out := map[string][]omsDomain.SalesOrder{}
	for _, o := range orders {
		if o.ConnectionID == nil {
			continue
		}
		out[*o.ConnectionID] = append(out[*o.ConnectionID], o)
	}
	return out
}
