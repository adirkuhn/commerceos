package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type pickListOrderFinder interface {
	FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

type pickListLineItemLister interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error)
}

// BuildPickListUseCase is the read-only pick list across several sales
// orders -- what to fetch off the shelves before packing them, for when the
// full picking module (sessions, scans) is skipped. Never writes anything;
// see domain/oms.BuildPickList for the aggregation rules.
type BuildPickListUseCase struct {
	db           *gorm.DB
	orderRepo    pickListOrderFinder
	lineItemRepo pickListLineItemLister
}

func NewBuildPickListUseCase(db *gorm.DB, orderRepo pickListOrderFinder, lineItemRepo pickListLineItemLister) *BuildPickListUseCase {
	return &BuildPickListUseCase{db: db, orderRepo: orderRepo, lineItemRepo: lineItemRepo}
}

func (uc *BuildPickListUseCase) Execute(ctx context.Context, salesOrderIDs []string) (*omsDomain.PickList, error) {
	ids := dedupeIDs(salesOrderIDs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("buildPickList: %w", omsDomain.ErrPickListOrderIDsRequired)
	}
	if len(ids) > omsDomain.MaxPickListOrders {
		return nil, fmt.Errorf("buildPickList: %w", omsDomain.ErrPickListTooManyOrders)
	}

	var orders []omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, id := range ids {
			order, err := uc.orderRepo.FindByID(ctx, id)
			if err != nil {
				return err
			}
			items, err := uc.lineItemRepo.ListBySalesOrder(ctx, id)
			if err != nil {
				return err
			}
			orders = append(orders, *order)
			lineItems = append(lineItems, items...)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("buildPickList: %w", err)
	}

	list := omsDomain.BuildPickList(orders, lineItems)
	return &list, nil
}

func dedupeIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
