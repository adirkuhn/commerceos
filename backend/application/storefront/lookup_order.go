package storefront

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// OrderStatus is the lean shape a guest order-status lookup returns --
// never the full SalesOrder (no internal ids, no full address, nothing a
// storefront key holder shouldn't see beyond "did this order go through
// and what's happening with it").
type OrderStatus struct {
	OrderNumber     string
	Status          omsDomain.SalesOrderStatus
	Fulfilled       bool
	TotalMinorUnits int64
	CurrencyCode    string
}

type LookupOrderUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
}

func NewLookupOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository) *LookupOrderUseCase {
	return &LookupOrderUseCase{db: db, salesOrderRepo: salesOrderRepo}
}

// Execute requires orderNumber AND email to match -- an order number alone
// is not proof of ownership (same "track my order" convention every
// storefront uses). Also requires the order to have been placed on
// salesChannelID specifically -- see SalesOrder.SalesChannelID's doc
// comment: a tenant's B2C key must never surface a B2B order's status, or
// vice versa.
func (uc *LookupOrderUseCase) Execute(ctx context.Context, salesChannelID, orderNumber, email string) (*OrderStatus, error) {
	var order *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.salesOrderRepo.FindByNumber(ctx, orderNumber)
		if err != nil {
			return err
		}
		order = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lookupOrder: %w", err)
	}
	if order == nil ||
		order.SalesChannelID == nil || *order.SalesChannelID != salesChannelID ||
		!strings.EqualFold(order.Billing.Email, email) {
		return nil, storefrontDomain.ErrOrderNotFound
	}
	return &OrderStatus{
		OrderNumber: order.Number, Status: order.Status, Fulfilled: order.Fulfilled,
		TotalMinorUnits: order.TotalMinorUnits, CurrencyCode: order.CurrencyCode,
	}, nil
}
