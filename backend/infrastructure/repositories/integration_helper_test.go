//go:build integration

// Integration tests in this package run against a real, ephemeral Postgres
// container (testcontainers-go) with every migration applied — unlike the
// rest of this package's tests, which mock the SQL layer with sqlmock.
// They're gated behind the "integration" build tag (needs Docker) and
// excluded from the default `go test ./...` run:
//
//	go test -tags=integration ./infrastructure/repositories/...
package repositories_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const migrationsDir = "../db/migrations"

// newIntegrationDB starts a fresh Postgres container, runs every migration
// against it, and returns a connected *gorm.DB. One container per test
// (not shared/reused) — slower, but guarantees no state leaks between
// tests, which matters more here than raw speed.
func newIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, _ := newIntegrationDBAndDSN(t)
	return db
}

// newIntegrationDBAndDSN is newIntegrationDB but also returns the
// superuser connection string, so a caller can derive a second connection
// as a different role (see newAppRoleDB) against the same database.
func newIntegrationDBAndDSN(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("commerceos_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		// Postgres logs "database system is ready to accept connections"
		// twice on first boot — once for the initdb-triggered internal
		// restart, once for real. Waiting for only the first occurrence
		// (this module's default) connects during that restart window and
		// gets a connection reset; the second occurrence is the real one.
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, goose.Up(sqlDB, migrationsDir))

	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, dsn
}

// newAppRoleDB connects to the same database as superDSN (a superuser
// connection string returned by newIntegrationDBAndDSN), but as
// commerceos_app -- the dedicated non-superuser, non-BYPASSRLS role
// created by migration 020_app_role_and_force_rls.sql. Production traffic
// runs as this role, not as the migration-owning superuser, so this is
// the only connection in this package that actually exercises RLS the way
// it behaves in production (see TestBatchRepo_Integration_RLSHidesOther-
// TenantsBatches).
func newAppRoleDB(t *testing.T, superDSN string) *gorm.DB {
	t.Helper()

	u, err := url.Parse(superDSN)
	require.NoError(t, err)
	u.User = url.UserPassword("commerceos_app", "changeme")

	sqlDB, err := goose.OpenDBWithDriver("postgres", u.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB
}

// newTestTenant returns a fresh random tenant id and a context carrying it
// — every integration test gets its own tenant so RLS isolation can be
// asserted without tests stepping on each other's rows even though they
// share (within a single test) the same database.
func newTestTenant() (string, context.Context) {
	id := idgen.NewV4()
	return id, tenant.WithTenantID(context.Background(), id)
}
