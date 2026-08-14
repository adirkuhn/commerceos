package repositories_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// TestConnectionRepo_Create_omitsEmptyIDLettingPostgresGenerateIt guards
// against a real production bug: the model's "id" column has a Postgres-side
// DEFAULT gen_random_uuid() but no matching gorm "default:" tag, so GORM
// didn't know to omit the Go zero value ("") from the INSERT column list —
// Postgres then rejected the literal empty string as an invalid UUID.
func TestConnectionRepo_Create_omitsEmptyIDLettingPostgresGenerateIt(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	// "id" must not appear in the INSERT column list — it must be left for
	// Postgres's own DEFAULT to generate.
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "integration_connections" \("tenant_id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("generated-uuid"))
	mock.ExpectCommit()

	repo := repositories.NewConnectionRepo(gormDB)
	ctx := txctx.WithTx(context.Background(), gormDB)

	conn := &integrationsDomain.Connection{
		TenantID:                "tenant-1",
		Provider:                integrationsDomain.ProviderWooCommerce,
		StoreURL:                "https://mystore.com",
		ConsumerKey:             "ck_abc",
		ConsumerSecretEncrypted: "encrypted",
		Status:                  integrationsDomain.StatusConnected,
	}
	err = repo.Create(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, "generated-uuid", conn.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionRepo_Update_persistsDefaultWarehouseID(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "integration_connections" SET .*"default_warehouse_id"=\$\d.*WHERE id = \$\d`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := repositories.NewConnectionRepo(gormDB)
	ctx := txctx.WithTx(context.Background(), gormDB)

	warehouseID := "wh-1"
	conn := &integrationsDomain.Connection{ID: "conn-1", DefaultWarehouseID: &warehouseID}
	err = repo.Update(ctx, conn)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
