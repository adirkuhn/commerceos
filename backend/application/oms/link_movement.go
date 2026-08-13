package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// linkLineItemMovement records that stockMovementID was caused by
// lineItemID -- its own tenant.WithTenant call, same composition rule as
// recordSaleMovementUC: never nest inside another transaction.
func linkLineItemMovement(ctx context.Context, db *gorm.DB, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, lineItemID, movementID string) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return linkRepo.Create(ctx, &omsDomain.SalesOrderLineItemStockMovement{
			TenantID: tenant.IDFromContext(ctx), SalesOrderLineItemID: lineItemID,
			StockMovementID: movementID, CreatedAt: time.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("linkLineItemMovement: %w", err)
	}
	return nil
}
