//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestCustomerRepo_Integration_CreateFindListUpdateDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customerRepo := repositories.NewCustomerRepo(db)

	var customerID string
	wooID := int64(99)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), WooCustomerID: &wooID, Email: "jane@example.com",
			FirstName: "Jane", LastName: "Doe", Username: "janedoe", Role: "customer",
			Billing: customersDomain.Address{
				FirstName: "Jane", LastName: "Doe", Address1: "123 Main St", Address2: "Apt 4",
				City: "Springfield", State: "IL", Postcode: "62701", Country: "US", Email: "jane@example.com", Phone: "555-0100",
			},
			Shipping: customersDomain.Address{
				FirstName: "Jane", LastName: "Doe", Address1: "456 Oak Ave", City: "Springfield", Country: "US",
			},
			IsPayingCustomer: true, AvatarURL: "https://example.com/avatar.png",
			DateCreated: now, DateModified: now,
		}
		if err := customerRepo.Create(ctx, &c); err != nil {
			return err
		}
		customerID = c.ID
		return nil
	})
	require.NoError(t, err)

	var found *customersDomain.Customer
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = customerRepo.FindByID(ctx, customerID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "jane@example.com", found.Email)
	require.Equal(t, int64(99), *found.WooCustomerID)
	require.Equal(t, "123 Main St", found.Billing.Address1)
	require.Equal(t, "Apt 4", found.Billing.Address2)
	require.Equal(t, "456 Oak Ave", found.Shipping.Address1)
	require.True(t, found.IsPayingCustomer)

	var listed []customersDomain.Customer
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = customerRepo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.NotEmpty(t, listed)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.FirstName = "Janet"
		found.IsPayingCustomer = false
		return customerRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = customerRepo.FindByID(ctx, customerID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "Janet", found.FirstName)
	require.False(t, found.IsPayingCustomer, "Update via a map must be able to clear a bool back to false")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return customerRepo.Delete(ctx, customerID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := customerRepo.FindByID(ctx, customerID)
		return err
	})
	require.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}

func TestCustomerRepo_Integration_FindByIDNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customerRepo := repositories.NewCustomerRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := customerRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}

func TestCustomerRepo_Integration_FindByWooCustomerID(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customerRepo := repositories.NewCustomerRepo(db)

	wooID := int64(4242)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), WooCustomerID: &wooID, Email: "woo@example.com",
			FirstName: "Woo", LastName: "Buyer", DateCreated: now, DateModified: now,
		}
		return customerRepo.Create(ctx, &c)
	})
	require.NoError(t, err)

	var found *customersDomain.Customer
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = customerRepo.FindByWooCustomerID(ctx, wooID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, "woo@example.com", found.Email)
}

func TestCustomerRepo_Integration_FindByWooCustomerIDReturnsNilNotErrorWhenMissing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customerRepo := repositories.NewCustomerRepo(db)

	var found *customersDomain.Customer
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = customerRepo.FindByWooCustomerID(ctx, 999999)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}
