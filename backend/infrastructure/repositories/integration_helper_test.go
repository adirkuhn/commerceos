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
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
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

const (
	migrationsDir = "../db/migrations"
	// templateDB is migrated once per test run; every test gets a copy.
	templateDB = "commerceos_test"
)

// shared is the one Postgres container a test run uses (started on first
// use, so tests that never touch the database never start it).
var shared struct {
	once      sync.Once
	err       error
	container *tcpostgres.PostgresContainer
	baseURL   *url.URL // superuser connection, database path left to the caller
	admin     *sql.DB  // connected to "postgres", issues CREATE/DROP DATABASE
}

// TestMain owns the shared container's lifecycle: it stops it after the run.
//
// Ryuk (testcontainers' cleanup sidecar) is off for this binary: its session
// is keyed on the parent process, so every package binary of one
// `go test ./...` shares it, and when another package (internal/db) finished
// Ryuk removed this package's container mid-run. Trade-off: a hard-killed run
// (panic, timeout, Ctrl-C) leaves the container behind.
func TestMain(m *testing.M) {
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		fmt.Fprintln(os.Stderr, "disable ryuk:", err)
		os.Exit(1)
	}
	code := m.Run()
	if shared.admin != nil {
		if err := shared.admin.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close admin connection:", err)
		}
	}
	if shared.container != nil {
		if err := shared.container.Terminate(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "stop shared Postgres:", err)
		}
	}
	os.Exit(code)
}

// newIntegrationDB returns a connection to a fresh, fully migrated
// database of its own: a copy of the migrated template (CREATE DATABASE
// ... TEMPLATE, a file copy -- ~0.1s instead of a container start plus 237
// migrations per test). Dropped when the test ends, so no state leaks
// between tests.
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
	startSharedPostgres()
	require.NoError(t, shared.err, "start shared Postgres")

	name := "t_" + strings.ReplaceAll(idgen.NewV4(), "-", "")
	_, err := shared.admin.Exec(fmt.Sprintf(`CREATE DATABASE %q TEMPLATE %q`, name, templateDB))
	require.NoError(t, err)
	// Registered first, so it runs last: after every connection below closed.
	t.Cleanup(func() {
		if _, err := shared.admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, name)); err != nil {
			t.Errorf("drop test database %s: %v", name, err)
		}
	})

	dsn := databaseURL(name)
	sqlDB, err := goose.OpenDBWithDriver("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, dsn
}

func databaseURL(name string) string {
	u := *shared.baseURL
	u.Path = "/" + name
	return u.String()
}

// startSharedPostgres starts the container and migrates the template, once.
func startSharedPostgres() {
	shared.once.Do(func() {
		ctx := context.Background()
		container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
			tcpostgres.WithDatabase(templateDB),
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
		if err != nil {
			shared.err = fmt.Errorf("start container: %w", err)
			return
		}
		shared.container = container

		dsn, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			shared.err = err
			return
		}
		if shared.baseURL, err = url.Parse(dsn); err != nil {
			shared.err = err
			return
		}

		// Migrate the template, then close every connection to it: Postgres
		// only copies a database nobody is connected to.
		migrateDB, err := goose.OpenDBWithDriver("postgres", dsn)
		if err != nil {
			shared.err = err
			return
		}
		err = goose.Up(migrateDB, migrationsDir)
		_ = migrateDB.Close()
		if err != nil {
			shared.err = fmt.Errorf("migrate template: %w", err)
			return
		}

		shared.admin, shared.err = goose.OpenDBWithDriver("postgres", databaseURL("postgres"))
	})
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
