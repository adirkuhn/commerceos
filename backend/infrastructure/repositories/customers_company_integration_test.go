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

func TestCustomerRepo_Integration_CompanyFieldsRoundTripAndClientNumbersCountUp(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewCustomerRepo(db)

	company := &customersDomain.Customer{Type: customersDomain.TypeCompany, CompanyName: "Claus Hoofddorp B.V.", KVKNumber: "34073293", VATNumber: "NL001234567B01"}
	person := &customersDomain.Customer{FirstName: "Ana"}
	var found *customersDomain.Customer
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, c := range []*customersDomain.Customer{company, person} {
			c.TenantID, c.DateCreated, c.DateModified = tenant.IDFromContext(ctx), time.Now(), time.Now()
			if err := repo.Create(ctx, c); err != nil {
				return err
			}
		}
		var err error
		found, err = repo.FindByID(ctx, company.ID)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, "1", company.ClientNumber)
	assert.Equal(t, "2", person.ClientNumber)
	assert.Equal(t, customersDomain.TypeCompany, found.Type)
	assert.Equal(t, "Claus Hoofddorp B.V.", found.CompanyName)
	assert.Equal(t, "34073293", found.KVKNumber)
	assert.Equal(t, "NL001234567B01", found.VATNumber)
	assert.Equal(t, "1", found.ClientNumber)
}

func TestCustomerRepo_Integration_ClientNumberIsUniquePerTenant(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewCustomerRepo(db)

	first := &customersDomain.Customer{FirstName: "Ana", ClientNumber: "3673"}
	second := &customersDomain.Customer{FirstName: "Bia"}
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, c := range []*customersDomain.Customer{first, second} {
			c.TenantID, c.DateCreated, c.DateModified = tenant.IDFromContext(ctx), time.Now(), time.Now()
			if err := repo.Create(ctx, c); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "3674", second.ClientNumber, "numbering continues after the highest number")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		second.ClientNumber = "3673"
		return repo.Update(ctx, second)
	})
	assert.ErrorIs(t, err, customersDomain.ErrDuplicateClientNumber)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		dup := &customersDomain.Customer{TenantID: tenant.IDFromContext(ctx), FirstName: "Caio", ClientNumber: "3673", DateCreated: time.Now(), DateModified: time.Now()}
		return repo.Create(ctx, dup)
	})
	assert.ErrorIs(t, err, customersDomain.ErrDuplicateClientNumber)
}

func TestCustomerRepo_Integration_SearchFindsCompaniesByNameNumberAndKVK(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewCustomerRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, c := range []*customersDomain.Customer{
			{Type: customersDomain.TypeCompany, CompanyName: "Restaurante Sabor B.V.", KVKNumber: "34073293", ClientNumber: "3286"},
			{FirstName: "Ana", LastName: "Souza"},
		} {
			c.TenantID, c.DateCreated, c.DateModified = tenant.IDFromContext(ctx), time.Now(), time.Now()
			if err := repo.Create(ctx, c); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, query := range []string{"sabor", "Restaurant Sabr", "3286", "34073293"} {
		var results []customersDomain.Customer
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			results, err = repo.Search(ctx, query, 10)
			return err
		})
		require.NoError(t, err)
		require.NotEmpty(t, results, query)
		assert.Equal(t, "Restaurante Sabor B.V.", results[0].CompanyName, query)
	}
}

// Migration 233 makes customers known only by a billing company companies
// and numbers every customer per tenant, oldest first.
func TestMigration233_Integration_backfillsCompaniesAndClientNumbers(t *testing.T) {
	db, dsn := newIntegrationDBAndDSN(t)
	tenantID, _ := newTestTenant()
	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, goose.DownTo(sqlDB, migrationsDir, 232))

	require.NoError(t, db.Exec(`INSERT INTO customers (tenant_id, first_name, billing_company, date_created, date_modified) VALUES
		(?, '', 'Sabor B.V.', now() - interval '2 day', now()),
		(?, 'Ana', 'Ana''s Shop', now() - interval '1 day', now())`, tenantID, tenantID).Error)
	require.NoError(t, goose.UpTo(sqlDB, migrationsDir, 233))

	type row struct{ CustomerType, CompanyName, ClientNumber string }
	var rows []row
	require.NoError(t, db.Raw(`SELECT customer_type, company_name, client_number FROM customers WHERE tenant_id = ? ORDER BY date_created`, tenantID).Scan(&rows).Error)
	assert.Equal(t, []row{{"company", "Sabor B.V.", "1"}, {"private", "", "2"}}, rows)
}
