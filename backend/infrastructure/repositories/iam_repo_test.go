package repositories_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// TestGroupRepo_Create_omitsEmptyIDLettingPostgresGenerateIt — same bug
// class as integrations_repo_test.go's connection test: the "id" column has
// a Postgres-side DEFAULT gen_random_uuid() but the model lacked a matching
// gorm "default:" tag, so GORM inserted the Go zero value ("") literally
// instead of omitting the column.
func TestGroupRepo_Create_omitsEmptyIDLettingPostgresGenerateIt(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "iam_groups" \("tenant_id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("generated-uuid"))
	mock.ExpectCommit()

	repo := repositories.NewGroupRepo(gormDB)
	ctx := txctx.WithTx(context.Background(), gormDB)

	group := &iamDomain.Group{
		TenantID:    "tenant-1",
		Name:        "Editors",
		Description: "desc",
	}
	err = repo.Create(ctx, group)
	require.NoError(t, err)
	require.Equal(t, "generated-uuid", group.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
