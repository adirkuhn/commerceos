//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSalesOrderLineItemRepo_Integration_WeightGramsRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	weightGrams := int64(350)
	var weighedID, unitID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-order-weighed-1", WarehouseID: warehouseID,
			Status: omsDomain.SalesOrderStatusCompleted, CurrencyCode: "EUR", TotalMinorUnits: 1725,
			OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}

		weighed := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID, Name: "Carne seca",
			Quantity: 1, WeightGrams: &weightGrams, PriceMinorUnits: 1225, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &weighed); err != nil {
			return err
		}
		weighedID = weighed.ID

		unit := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID, Name: "Widget",
			Quantity: 1, PriceMinorUnits: 500, CreatedAt: now,
		}
		if err := lineItemRepo.Create(ctx, &unit); err != nil {
			return err
		}
		unitID = unit.ID
		return nil
	})
	require.NoError(t, err)

	var weighed, unit *omsDomain.SalesOrderLineItem
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if weighed, err = lineItemRepo.FindByID(ctx, weighedID); err != nil {
			return err
		}
		unit, err = lineItemRepo.FindByID(ctx, unitID)
		return err
	})
	require.NoError(t, err)

	require.NotNil(t, weighed.WeightGrams)
	assert.Equal(t, weightGrams, *weighed.WeightGrams)
	assert.Nil(t, unit.WeightGrams, "a unit line has no weight -- must round-trip nil, not 0")
}
