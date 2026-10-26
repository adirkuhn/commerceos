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
)

func TestCustomerContactPersonRepo_Integration_CreateListUpdateDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customers := repositories.NewCustomerRepo(db)
	repo := repositories.NewCustomerContactPersonRepo(db)

	company := &customersDomain.Customer{Type: customersDomain.TypeCompany, CompanyName: "Claus Hoofddorp B.V."}
	kylian := &customersDomain.ContactPerson{FirstName: "Kylian", LastName: "Klockenbrink", Role: "Employee"}
	sara := &customersDomain.ContactPerson{FirstName: "Sara", Email: "sara@claus.nl"}
	var listed []customersDomain.ContactPerson
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		company.TenantID, company.DateCreated, company.DateModified = tenant.IDFromContext(ctx), now, now
		if err := customers.Create(ctx, company); err != nil {
			return err
		}
		for i, p := range []*customersDomain.ContactPerson{kylian, sara} {
			p.TenantID, p.CustomerID = tenant.IDFromContext(ctx), company.ID
			p.CreatedAt, p.UpdatedAt = now.Add(time.Duration(i)*time.Second), now
			if err := repo.Create(ctx, p); err != nil {
				return err
			}
		}
		kylian.Phone, kylian.UpdatedAt = "0612345678", time.Now()
		if err := repo.Update(ctx, kylian); err != nil {
			return err
		}
		if err := repo.Delete(ctx, company.ID, sara.ID); err != nil {
			return err
		}
		var err error
		listed, err = repo.ListByCustomer(ctx, company.ID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "Kylian", listed[0].FirstName)
	assert.Equal(t, "Employee", listed[0].Role)
	assert.Equal(t, "0612345678", listed[0].Phone)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000", kylian.ID)
		return err
	})
	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNotFound, "a person is only found under its own customer")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error { return repo.Delete(ctx, company.ID, sara.ID) })
	assert.ErrorIs(t, err, customersDomain.ErrContactPersonNotFound)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := customers.Delete(ctx, company.ID); err != nil {
			return err
		}
		var err error
		listed, err = repo.ListByCustomer(ctx, company.ID)
		return err
	})
	require.NoError(t, err)
	assert.Empty(t, listed, "contact persons go with their customer")
}

func TestCustomerContactPersonRepo_Integration_RLSHidesOtherTenants(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	app := newAppRoleDB(t, dsn)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	repo := repositories.NewCustomerContactPersonRepo(app)

	var customerID string
	err := tenant.WithTenant(ctxA, app, func(ctx context.Context) error {
		c := &customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Type: customersDomain.TypeCompany, CompanyName: "A B.V.", DateCreated: time.Now(), DateModified: time.Now()}
		if err := repositories.NewCustomerRepo(app).Create(ctx, c); err != nil {
			return err
		}
		customerID = c.ID
		return repo.Create(ctx, &customersDomain.ContactPerson{TenantID: tenant.IDFromContext(ctx), CustomerID: c.ID, FirstName: "Ana", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	})
	require.NoError(t, err)

	var listed []customersDomain.ContactPerson
	err = tenant.WithTenant(ctxB, app, func(ctx context.Context) error {
		var err error
		listed, err = repo.ListByCustomer(ctx, customerID)
		return err
	})
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestCustomerRepo_Integration_SearchFindsACompanyByItsContactPerson(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	customers := repositories.NewCustomerRepo(db)

	var results []customersDomain.Customer
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c := &customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), Type: customersDomain.TypeCompany, CompanyName: "Claus Hoofddorp B.V.", DateCreated: time.Now(), DateModified: time.Now()}
		if err := customers.Create(ctx, c); err != nil {
			return err
		}
		if err := repositories.NewCustomerContactPersonRepo(db).Create(ctx, &customersDomain.ContactPerson{
			TenantID: tenant.IDFromContext(ctx), CustomerID: c.ID, FirstName: "Kylian", LastName: "Klockenbrink", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}); err != nil {
			return err
		}
		var err error
		results, err = customers.Search(ctx, "klockenbrink", 10)
		return err
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Claus Hoofddorp B.V.", results[0].CompanyName)
}
