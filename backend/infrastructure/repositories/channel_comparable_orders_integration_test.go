//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSalesOrderRepo_Integration_ListChannelComparableOrders(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	repo := repositories.NewSalesOrderRepo(db)
	connID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderWooCommerce)
	otherConnID := seedConnectionOf(t, db, ctx, integrationsDomain.ProviderBol)
	ids := map[string]string{}
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i, s := range []struct {
			extID  string
			conn   *string
			status omsDomain.SalesOrderStatus
		}{
			{"cmp-completed", &connID, omsDomain.SalesOrderStatusCompleted},
			{"cmp-processing", &connID, omsDomain.SalesOrderStatusProcessing},
			{"cmp-cancelled", &connID, omsDomain.SalesOrderStatusCancelled},
			{"cmp-refunded", &connID, omsDomain.SalesOrderStatusRefunded},
			{"cmp-onhold", &connID, omsDomain.SalesOrderStatusOnHold},
			{"cmp-other-connection", &otherConnID, omsDomain.SalesOrderStatusProcessing},
		} {
			at := base.Add(time.Duration(i) * time.Minute)
			o := omsDomain.SalesOrder{
				TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: s.extID,
				ConnectionID: s.conn, WarehouseID: warehouseID, CurrencyCode: "EUR", Status: s.status, OccurredAt: at, CreatedAt: at,
			}
			if err := repo.Create(ctx, &o); err != nil {
				return err
			}
			ids[s.extID] = o.ID
		}
		return nil
	}))

	list := func(after *omsDomain.OrderPageCursor, limit int) []string {
		var found []omsDomain.SalesOrder
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = repo.ListChannelComparableOrders(ctx, connID, after, limit)
			return err
		}))
		out := make([]string, len(found))
		for i, o := range found {
			out[i] = o.ExternalOrderID
		}
		return out
	}

	assert.Equal(t, []string{"cmp-onhold", "cmp-processing", "cmp-completed"}, list(nil, 50), "newest first, never-shipping statuses and other connections left out")
	assert.Equal(t, []string{"cmp-onhold", "cmp-processing"}, list(nil, 2))
	cursor := &omsDomain.OrderPageCursor{OccurredAt: base.Add(time.Minute), ID: ids["cmp-processing"]}
	assert.Equal(t, []string{"cmp-completed"}, list(cursor, 2))
}
