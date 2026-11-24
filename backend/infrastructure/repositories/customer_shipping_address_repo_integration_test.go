//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestShippingAddressRepo_Integration_bookLifecycle(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	tenantID, ctx := newTestTenant()
	customers := repositories.NewCustomerRepo(db)
	repo := repositories.NewCustomerShippingAddressRepo(db)
	now := time.Now()

	var customerID string
	var book customersDomain.ShippingBook
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c := &customersDomain.Customer{TenantID: tenantID, FirstName: "Ana", DateCreated: now, DateModified: now}
		if err := customers.Create(ctx, c); err != nil {
			return err
		}
		customerID = c.ID
		for i, line := range []string{"Main 1", "Dock 2"} {
			a := &customersDomain.ShippingAddress{
				TenantID: tenantID, CustomerID: customerID, Label: line, IsDefault: i == 0, Position: i,
				Address: customersDomain.Address{FirstName: "Ana", Address1: line, City: "Utrecht", Postcode: "3511", Country: "NL", Phone: "06"},
				CreatedAt: now, UpdatedAt: now,
			}
			if err := repo.Create(ctx, a); err != nil {
				return err
			}
		}
		var err error
		book, err = repo.ListByCustomer(ctx, customerID)
		return err
	}))
	require.Len(t, book, 2)
	assert.Equal(t, "Main 1", book[0].Address.Address1)
	assert.Equal(t, "3511", book[0].Address.Postcode)
	assert.True(t, book[0].IsDefault)
	assert.Equal(t, "Dock 2", book[1].Label)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := repo.SetDefault(ctx, customerID, book[1].ID); err != nil {
			return err
		}
		book[0].Address.City = "Delft"
		if err := repo.Update(ctx, &book[0]); err != nil {
			return err
		}
		var err error
		book, err = repo.ListByCustomer(ctx, customerID)
		return err
	}))
	assert.False(t, book[0].IsDefault)
	assert.True(t, book[1].IsDefault, "only one default")
	assert.Equal(t, "Delft", book[0].Address.City)

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := repo.Delete(ctx, customerID, book[0].ID); err != nil {
			return err
		}
		assert.ErrorIs(t, repo.Delete(ctx, customerID, book[0].ID), customersDomain.ErrShippingAddressNotFound)
		assert.ErrorIs(t, repo.Update(ctx, &book[0]), customersDomain.ErrShippingAddressNotFound)
		assert.ErrorIs(t, repo.SetDefault(ctx, customerID, book[0].ID), customersDomain.ErrShippingAddressNotFound)
		var err error
		book, err = repo.ListByCustomer(ctx, customerID)
		return err
	}))
	require.Len(t, book, 1)

	// Another tenant sees nothing (RLS).
	_, otherCtx := newTestTenant()
	app := newAppRoleDB(t, dsn)
	require.NoError(t, tenant.WithTenant(otherCtx, app, func(ctx context.Context) error {
		other, err := repositories.NewCustomerShippingAddressRepo(app).ListByCustomer(ctx, customerID)
		assert.Empty(t, other)
		return err
	}))
}

// Creating a customer stores a given shipping address as their default,
// so every way a customer is made (admin, storefront, WhatsApp, Woo
// import) starts with a book.
func TestCustomerRepo_Integration_createSeedsTheDefaultShippingAddress(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	customers := repositories.NewCustomerRepo(db)
	repo := repositories.NewCustomerShippingAddressRepo(db)
	now := time.Now()

	var withAddress, without, toBilling string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, c := range []struct {
			id            *string
			shipping      customersDomain.Address
			shipToBilling bool
		}{
			{&withAddress, customersDomain.Address{Address1: "Main 1", City: "Utrecht"}, false},
			{&without, customersDomain.Address{}, false},
			{&toBilling, customersDomain.Address{Address1: "Bill 1"}, true},
		} {
			cust := &customersDomain.Customer{TenantID: tenantID, FirstName: "Ana", Shipping: c.shipping, ShipToBilling: c.shipToBilling, DateCreated: now, DateModified: now}
			if err := customers.Create(ctx, cust); err != nil {
				return err
			}
			*c.id = cust.ID
		}
		return nil
	}))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		book, err := repo.ListByCustomer(ctx, withAddress)
		require.NoError(t, err)
		require.Len(t, book, 1)
		assert.True(t, book[0].IsDefault)
		assert.Equal(t, "Main 1", book[0].Address.Address1)

		book, err = repo.ListByCustomer(ctx, without)
		require.NoError(t, err)
		assert.Empty(t, book)

		book, err = repo.ListByCustomer(ctx, toBilling)
		require.NoError(t, err)
		assert.Empty(t, book, "ships to billing: the copy isn't an address of its own")

		found, err := customers.FindByID(ctx, toBilling)
		require.NoError(t, err)
		assert.True(t, found.ShipToBilling)

		found.ShipToBilling = false
		require.NoError(t, customers.Update(ctx, found))
		found, err = customers.FindByID(ctx, toBilling)
		require.NoError(t, err)
		assert.False(t, found.ShipToBilling, "can be switched off again")
		return nil
	}))
}

// Migration 253 turns every customer's shipping address into their
// default one.
func TestMigration253_Integration_existingShippingBecomesTheDefault(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	tenantID, ctx := newTestTenant()
	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 252))
	t.Cleanup(func() { _ = goose.Up(sqlDB, migrationsDir) })

	var withShipping, withoutShipping string
	require.NoError(t, db.Raw(`INSERT INTO customers (tenant_id, first_name, client_number, shipping_address_1, shipping_city, date_created, date_modified)
		VALUES (?, 'Ana', 'm-1', 'Main 1', 'Utrecht', now(), now()) RETURNING id`, tenantID).Scan(&withShipping).Error)
	require.NoError(t, db.Raw(`INSERT INTO customers (tenant_id, first_name, client_number, date_created, date_modified)
		VALUES (?, 'Bo', 'm-2', now(), now()) RETURNING id`, tenantID).Scan(&withoutShipping).Error)

	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 253))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		repo := repositories.NewCustomerShippingAddressRepo(db)
		book, err := repo.ListByCustomer(ctx, withShipping)
		require.NoError(t, err)
		require.Len(t, book, 1)
		assert.True(t, book[0].IsDefault)
		assert.Equal(t, "Main 1", book[0].Address.Address1)
		assert.Equal(t, "Utrecht", book[0].Address.City)

		book, err = repo.ListByCustomer(ctx, withoutShipping)
		require.NoError(t, err)
		assert.Empty(t, book)
		return nil
	}))
}
