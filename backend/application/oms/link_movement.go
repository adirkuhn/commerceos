package oms

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// linkLineItemMovement records that movement was caused by lineItemID --
// its own tenant.WithTenant call, same composition rule as
// recordSaleMovementUC: never nest inside another transaction. Denormalizes
// movement's BatchID/QuantityDelta onto the link row so a later restock can
// reverse exactly what this movement did without a second lookup -- see
// SalesOrderLineItemStockMovement's doc.
func linkLineItemMovement(ctx context.Context, db *gorm.DB, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, lineItemID string, movement invDomain.StockMovement) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return linkRepo.Create(ctx, &omsDomain.SalesOrderLineItemStockMovement{
			TenantID: tenant.IDFromContext(ctx), SalesOrderLineItemID: lineItemID,
			StockMovementID: movement.ID, BatchID: movement.BatchID, QuantityDelta: movement.QuantityDelta,
			CreatedAt: time.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("linkLineItemMovement: %w", err)
	}
	return nil
}
