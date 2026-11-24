//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// A customer is paying when one of their orders makes them one
// (omsDomain.SalesOrder.MakesPayingCustomer) -- worked out on every read,
// never stored, so it can't drift from the orders.
func TestCustomerRepo_Integration_payingCustomerComesFromTheOrders(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	repo := repositories.NewCustomerRepo(db)
	paidAt := time.Now()

	customers := map[string]struct {
		status   string
		datePaid *time.Time
		paying   bool
	}{
		"no-orders":  {"", nil, false},
		"pending":    {"pending", nil, false},
		"processing": {"processing", nil, true},
		"paid-date":  {"pending", &paidAt, true},
		"partial":    {"partially_refunded", &paidAt, true},
		"refunded":   {"refunded", &paidAt, false},
		"failed":     {"failed", &paidAt, false},
		"cancelled":  {"cancelled", &paidAt, false},
	}
	ids := map[string]string{}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for name, c := range customers {
			cust := &customersDomain.Customer{
				TenantID: tenantID, FirstName: "Paying", LastName: name, Email: name + "@example.com",
				DateCreated: paidAt, DateModified: paidAt,
			}
			if err := repo.Create(ctx, cust); err != nil {
				return err
			}
			ids[name] = cust.ID
			if c.status != "" {
				insertCustomerOrder(t, ctx, tenantID, warehouseID, cust.ID, c.status, c.datePaid)
			}
		}
		return nil
	}))

	var listed, searched []customersDomain.Customer
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if listed, err = repo.List(ctx); err != nil {
			return err
		}
		searched, err = repo.Search(ctx, "Paying", 50)
		return err
	}))
	byID := func(list []customersDomain.Customer) map[string]bool {
		out := map[string]bool{}
		for _, c := range list {
			out[c.ID] = c.IsPayingCustomer
		}
		return out
	}
	listedPaying, searchedPaying := byID(listed), byID(searched)

	for name, c := range customers {
		var found *customersDomain.Customer
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = repo.FindByID(ctx, ids[name])
			return err
		}))
		assert.Equal(t, c.paying, found.IsPayingCustomer, "FindByID %s", name)
		assert.Equal(t, c.paying, listedPaying[ids[name]], "List %s", name)
		assert.Equal(t, c.paying, searchedPaying[ids[name]], "Search %s", name)
	}

	// The other lookups read it too.
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByEmail(ctx, "processing@example.com")
		require.NoError(t, err)
		assert.True(t, found.IsPayingCustomer, "FindByEmail")
		return nil
	}))
}

func insertCustomerOrder(t *testing.T, ctx context.Context, tenantID, warehouseID, customerID, status string, datePaid *time.Time) {
	t.Helper()
	tx, ok := txctx.FromContext(ctx)
	require.True(t, ok, "call inside tenant.WithTenant")
	now := time.Now()
	require.NoError(t, tx.Exec(`INSERT INTO sales_orders (tenant_id, number, warehouse_id, currency_code, status, customer_id, date_paid,
			occurred_at, created_at, updated_at)
		VALUES (?, ?, ?, 'EUR', ?, ?, ?, ?, ?, ?)`,
		tenantID, "T-"+customerID[:8], warehouseID, status, customerID, datePaid, now, now, now).Error)
}
