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

// TestCustomerRepo_Integration_SearchByNameIsAccentInsensitiveAndTypoTolerant
// proves the same unaccent()/pg_trgm search Products already has
// (127_search_trigram_indexes.sql) actually works against real Postgres
// for customers too -- powers global search (Cmd+K).
func TestCustomerRepo_Integration_SearchByNameIsAccentInsensitiveAndTypoTolerant(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewCustomerRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for _, names := range [][2]string{{"João", "Conceição"}, {"Renée", "Müller"}} {
			c := &customersDomain.Customer{
				TenantID: tenant.IDFromContext(ctx), Email: names[0] + "@example.com",
				FirstName: names[0], LastName: names[1], DateCreated: now, DateModified: now,
			}
			if err := repo.Create(ctx, c); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		query    string
		wantName string
	}{
		{"joao conceicao", "João"},
		{"JOAO", "João"},
		{"Conceicoa", "João"}, // typo, transposed letters
		{"renee muller", "Renée"},
	} {
		var results []customersDomain.Customer
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			results, err = repo.Search(ctx, tc.query, 10)
			return err
		})
		require.NoError(t, err)
		require.Len(t, results, 1, "query %q should match exactly one customer", tc.query)
		require.Equal(t, tc.wantName, results[0].FirstName, "query %q", tc.query)
	}
}

func TestCustomerRepo_Integration_SearchByEmail(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewCustomerRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := &customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), Email: "unique-lookup@example.com",
			FirstName: "Pat", LastName: "Smith", DateCreated: now, DateModified: now,
		}
		return repo.Create(ctx, c)
	})
	require.NoError(t, err)

	var results []customersDomain.Customer
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, err = repo.Search(ctx, "unique-lookup", 10)
		return err
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "unique-lookup@example.com", results[0].Email)
}

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
			AvatarURL: "https://example.com/avatar.png", CustomerNote: "VIP customer",
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
	require.False(t, found.IsPayingCustomer, "no orders yet")
	require.Equal(t, "VIP customer", found.CustomerNote)

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
		found.CustomerNote = "Call before delivery"
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
	require.Equal(t, "Call before delivery", found.CustomerNote)

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

func TestCustomerRepo_Integration_FindByEmailIsCaseInsensitive(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customerRepo := repositories.NewCustomerRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), Email: "Jane@Example.com",
			FirstName: "Jane", LastName: "Doe", DateCreated: now, DateModified: now,
		}
		return customerRepo.Create(ctx, &c)
	})
	require.NoError(t, err)

	var found *customersDomain.Customer
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = customerRepo.FindByEmail(ctx, "jane@example.com")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, "Jane", found.FirstName)
}

func TestCustomerRepo_Integration_FindByEmailReturnsNilNotErrorWhenMissing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customerRepo := repositories.NewCustomerRepo(db)

	var found *customersDomain.Customer
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = customerRepo.FindByEmail(ctx, "nobody@example.com")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
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
