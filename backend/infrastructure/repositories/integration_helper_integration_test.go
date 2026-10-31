//go:build integration

package repositories_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func currentDatabase(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var name string
	require.NoError(t, db.Raw("SELECT current_database()").Scan(&name).Error)
	return name
}

// The helper's contract every integration test relies on: each call gets
// its own fully migrated database, so nothing one test writes is visible to
// another.
func TestIntegrationHelper_EachCallGetsItsOwnMigratedDatabase(t *testing.T) {
	first := newIntegrationDB(t)
	second := newIntegrationDB(t)

	for _, db := range []*gorm.DB{first, second} {
		var latest int64
		require.NoError(t, db.Raw("SELECT max(version_id) FROM goose_db_version").Scan(&latest).Error)
		assert.GreaterOrEqual(t, latest, int64(237), "every migration applied")
	}

	require.NoError(t, first.Exec("CREATE TABLE helper_isolation_probe (id int)").Error)
	var visible bool
	require.NoError(t, second.Raw("SELECT to_regclass('helper_isolation_probe') IS NOT NULL").Scan(&visible).Error)
	assert.False(t, visible, "a table made in one test's database must not exist in another's")
}

// newAppRoleDB must reach the same database as the superuser connection it
// was derived from, as the non-superuser role RLS applies to.
func TestIntegrationHelper_AppRoleConnectsToTheSameDatabase(t *testing.T) {
	super, dsn := newIntegrationDBAndDSN(t)
	app := newAppRoleDB(t, dsn)

	assert.Equal(t, currentDatabase(t, super), currentDatabase(t, app))
	var role string
	require.NoError(t, app.Raw("SELECT current_user").Scan(&role).Error)
	assert.Equal(t, "commerceos_app", role)
}
