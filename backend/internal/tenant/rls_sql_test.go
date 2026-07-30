package tenant_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestWithTenant_usesSetConfigWithBoundParam locks in the exact SQL shape
// WithTenant must emit to set the RLS tenant. Postgres's SET/SET LOCAL only
// accepts a literal value, never a bind parameter ("SET LOCAL x = $1" is a
// syntax error) — set_config() is a normal function call and does accept
// one. sqlmock verifies WHICH query text our code sends; it does not itself
// validate Postgres SQL syntax (it has no real parser), so this test proves
// intent/regression-lock, not independently that Postgres accepts it.
func TestWithTenant_usesSetConfigWithBoundParam(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config\('app.current_tenant', \$1, true\)`).
		WithArgs("tenant-123").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	ctx := tenant.WithTenantID(context.Background(), "tenant-123")
	err = tenant.WithTenant(ctx, gormDB, func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWithTenant_bypassRLSAlsoUsesSetConfig(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config\('app.current_tenant', \$1, true\)`).
		WithArgs("bypass_rls").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	ctx := tenant.WithSuperAdmin(context.Background())
	err = tenant.WithTenant(ctx, gormDB, func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
