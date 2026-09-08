package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// AccountOrder is the lean per-order shape an account order-history list
// shows -- same fields as OrderStatus (guest lookup), plus the order date,
// never the full SalesOrder.
type AccountOrder struct {
	OrderNumber     string
	Status          omsDomain.SalesOrderStatus
	Fulfilled       bool
	TotalMinorUnits int64
	CurrencyCode    string
	PlacedAt        time.Time
}

type ListAccountOrdersUseCase struct {
	db             *gorm.DB
	sessionRepo    storefrontDomain.CustomerSessionRepository
	salesOrderRepo omsDomain.SalesOrderRepository
}

func NewListAccountOrdersUseCase(db *gorm.DB, sessionRepo storefrontDomain.CustomerSessionRepository, salesOrderRepo omsDomain.SalesOrderRepository) *ListAccountOrdersUseCase {
	return &ListAccountOrdersUseCase{db: db, sessionRepo: sessionRepo, salesOrderRepo: salesOrderRepo}
}

// Execute lists sessionToken's customer's orders placed on salesChannelID
// specifically -- never the customer's whole cross-channel history, same
// channel-isolation rule LookupOrderUseCase's own doc comment enforces for
// guest order-status lookup.
func (uc *ListAccountOrdersUseCase) Execute(ctx context.Context, salesChannelID, sessionToken string) ([]AccountOrder, error) {
	var orders []AccountOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := resolveSession(ctx, uc.sessionRepo, salesChannelID, sessionToken)
		if err != nil {
			return err
		}
		found, _, err := uc.salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
			CustomerID: &session.CustomerID, SalesChannelID: &salesChannelID,
		})
		if err != nil {
			return err
		}
		orders = make([]AccountOrder, len(found))
		for i, o := range found {
			orders[i] = AccountOrder{
				OrderNumber: o.Number, Status: o.Status, Fulfilled: o.Fulfilled,
				TotalMinorUnits: o.TotalMinorUnits, CurrencyCode: o.CurrencyCode, PlacedAt: o.CreatedAt,
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listAccountOrders: %w", err)
	}
	return orders, nil
}
