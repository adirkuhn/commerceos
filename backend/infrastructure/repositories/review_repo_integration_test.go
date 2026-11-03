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

// A "verified purchase" review badge needs a completed order, on the same
// storefront channel, for the same product, billed to the reviewer's email
// (case-insensitive) -- anything short of completed (not delivered yet) or
// any mismatch is not verified.
func TestReviewRepo_Integration_HasVerifiedPurchase(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	channelID := seedInternalSalesChannel(t, db, ctx)
	otherChannelID := seedInternalSalesChannel(t, db, ctx)
	orderRepo := repositories.NewSalesOrderRepo(db)
	lineRepo := repositories.NewSalesOrderLineItemRepo(db)
	reviewRepo := repositories.NewReviewRepo(db)

	var productID string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return db.WithContext(ctx).Raw(`SELECT v.product_id FROM skus s JOIN variants v ON v.id = s.variant_id WHERE s.id = ?`, skuID).Scan(&productID).Error
	}))
	require.NotEmpty(t, productID)

	seed := func(extID, email string, status omsDomain.SalesOrderStatus, channel string) {
		now := time.Now()
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			o := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), 
				WarehouseID: warehouseID, Status: status, CurrencyCode: "EUR", SalesChannelID: &channel,
				Billing: omsDomain.Address{Email: email}, TotalMinorUnits: 1000, OccurredAt: now, CreatedAt: now,
			}
			if err := orderRepo.Create(ctx, &o); err != nil {
				return err
			}
			return lineRepo.Create(ctx, &omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: o.ID, SKUID: &skuID, Name: "P", Quantity: 1,
				PriceMinorUnits: 1000, TotalMinorUnits: 1000, CreatedAt: now,
			})
		}))
	}
	seed("rev-done", "Buyer@Example.com", omsDomain.SalesOrderStatusCompleted, channelID)
	seed("rev-shipped", "shipped@example.com", omsDomain.SalesOrderStatusShipped, channelID)
	seed("rev-other-channel", "elsewhere@example.com", omsDomain.SalesOrderStatusCompleted, otherChannelID)

	check := func(channel, email string) bool {
		var ok bool
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			ok, err = reviewRepo.HasVerifiedPurchase(ctx, channel, productID, email)
			return err
		}))
		return ok
	}
	assert.True(t, check(channelID, "buyer@example.com"), "completed, email matched case-insensitively")
	assert.False(t, check(channelID, "shipped@example.com"), "not completed yet")
	assert.False(t, check(channelID, "elsewhere@example.com"), "bought on another channel")
	assert.False(t, check(channelID, "stranger@example.com"), "never bought")
}
