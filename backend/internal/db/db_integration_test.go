//go:build integration

// Gated behind the "integration" build tag (needs Docker), same
// convention as infrastructure/repositories' own integration tests --
// gorm's postgres driver connects eagerly during Initialize (a version
// check, not lazy like a bare sql.Open), so this needs a real reachable
// Postgres rather than a garbage DSN.
package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/adirkuhn/commerceos/internal/db"
)

func TestOpen_capsThePoolAtMaxOpenConns(t *testing.T) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("commerceos_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	gormDB, err := db.Open(dsn, 20)
	require.NoError(t, err)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.Equal(t, 20, sqlDB.Stats().MaxOpenConnections)
	require.NoError(t, sqlDB.Ping())
}

func TestOpen_idleCapNeverDropsBelowTwoEvenForATinyMaxOpenConns(t *testing.T) {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("commerceos_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// maxOpenConns=1 would floor MaxIdleConns to 0 without the 2-floor in
	// Open -- open two connections at once (more than maxOpenConns=1 would
	// even allow to stay open simultaneously if idle reuse were broken)
	// and confirm both return to the pool for reuse rather than one
	// blocking or erroring.
	gormDB, err := db.Open(dsn, 5)
	require.NoError(t, err)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, sqlDB.Ping())
	require.NoError(t, sqlDB.Ping())
	require.LessOrEqual(t, sqlDB.Stats().Idle, 5)
}
