package theme_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func newDraft(t *testing.T) (theme.Theme, theme.Version) {
	t.Helper()
	th, err := theme.NewTheme("tenant-1", "chan-1", "Autumn", "default", now)
	require.NoError(t, err)
	th.ID = "theme-1"
	v := theme.NewDraftVersion(th, 1, theme.DefaultSettings(), "user-1", now)
	v.ID = "v-1"
	return th, v
}

func TestNewDraftVersion(t *testing.T) {
	_, v := newDraft(t)

	assert.Equal(t, theme.VersionStateDraft, v.State)
	assert.Equal(t, 1, v.Number)
	assert.Equal(t, 1, v.Revision)
	assert.Equal(t, "theme-1", v.ThemeID)
	assert.True(t, v.IsDraft())
}

func TestVersion_BeginEdit_bumpsRevision(t *testing.T) {
	_, v := newDraft(t)

	require.NoError(t, v.BeginEdit(1))
	assert.Equal(t, 2, v.Revision)
}

func TestVersion_BeginEdit_staleRevisionConflicts(t *testing.T) {
	_, v := newDraft(t)
	require.NoError(t, v.BeginEdit(1))

	assert.ErrorIs(t, v.BeginEdit(1), theme.ErrRevisionConflict)
	assert.Equal(t, 2, v.Revision)
}

func TestVersion_sealedIsImmutable(t *testing.T) {
	th, v := newDraft(t)
	require.NoError(t, theme.Publish(&th, &v, 1, "first", now))

	assert.ErrorIs(t, v.BeginEdit(v.Revision), theme.ErrVersionSealed)
	assert.ErrorIs(t, theme.Publish(&th, &v, v.Revision, "again", now), theme.ErrVersionSealed)
}

func TestPublish_sealsDraftAndMakesThemeLive(t *testing.T) {
	th, v := newDraft(t)
	later := now.Add(time.Minute)

	require.NoError(t, theme.Publish(&th, &v, 1, "launch", later))

	assert.Equal(t, theme.VersionStateSealed, v.State)
	require.NotNil(t, v.SealedAt)
	assert.Equal(t, later, *v.SealedAt)
	assert.Equal(t, "launch", v.Note)
	assert.True(t, th.IsLive)
	require.NotNil(t, th.PublishedVersionID)
	assert.Equal(t, "v-1", *th.PublishedVersionID)
}

func TestPublish_staleRevisionLeavesEverythingUntouched(t *testing.T) {
	th, v := newDraft(t)
	require.NoError(t, v.BeginEdit(1))

	err := theme.Publish(&th, &v, 1, "launch", now)

	assert.ErrorIs(t, err, theme.ErrRevisionConflict)
	assert.True(t, v.IsDraft())
	assert.False(t, th.IsLive)
	assert.Nil(t, th.PublishedVersionID)
}

func TestPublish_rejectsVersionOfAnotherTheme(t *testing.T) {
	th, v := newDraft(t)
	v.ThemeID = "other"

	assert.ErrorIs(t, theme.Publish(&th, &v, 1, "", now), theme.ErrVersionNotInTheme)
}

func TestRollback_pointsLiveAtOlderSealedVersion(t *testing.T) {
	th, v1 := newDraft(t)
	require.NoError(t, theme.Publish(&th, &v1, 1, "", now))
	v2 := theme.NewDraftVersion(th, 2, theme.DefaultSettings(), "user-1", now)
	v2.ID = "v-2"
	require.NoError(t, theme.Publish(&th, &v2, 1, "", now))

	require.NoError(t, theme.Rollback(&th, v1, now))

	assert.Equal(t, "v-1", *th.PublishedVersionID)
	assert.True(t, th.IsLive)
}

func TestRollback_rejectsDraftsAndForeignVersions(t *testing.T) {
	th, v := newDraft(t)
	assert.ErrorIs(t, theme.Rollback(&th, v, now), theme.ErrVersionNotSealed)

	require.NoError(t, theme.Publish(&th, &v, 1, "", now))
	v.ThemeID = "other"
	assert.ErrorIs(t, theme.Rollback(&th, v, now), theme.ErrVersionNotInTheme)
}

func TestNextDraftFrom_copiesSettingsIntoNewNumber(t *testing.T) {
	th, v := newDraft(t)
	v.Settings.Colors.Primary = "#112233"
	require.NoError(t, theme.Publish(&th, &v, 1, "", now))

	next := theme.NextDraftFrom(v, 2, "user-2", now)

	assert.Equal(t, 2, next.Number)
	assert.True(t, next.IsDraft())
	assert.Equal(t, 1, next.Revision)
	assert.Equal(t, "#112233", next.Settings.Colors.Primary)
	assert.Equal(t, "user-2", next.CreatedBy)
	assert.Empty(t, next.ID)
}
