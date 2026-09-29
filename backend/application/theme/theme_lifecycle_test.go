package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestCreateTheme_createsDraftWithPresetDocumentsAndRaisesEvent(t *testing.T) {
	w := newWorld(t)

	th := w.createTheme(t)

	assert.Equal(t, "Autumn", th.Name)
	assert.Equal(t, "tenant-1", th.TenantID)
	assert.False(t, th.IsLive)
	draft := w.draft(t, th.ID)
	assert.Equal(t, 1, draft.Number)
	assert.Equal(t, "user-1", draft.CreatedBy)
	assert.Equal(t, theme.DefaultSettings(), draft.Settings)
	preset, _ := theme.PresetDocuments(theme.PresetDefault)
	assert.Len(t, w.docs.forVersion(draft.ID), len(preset))
	assert.Equal(t, []string{"storefront.theme.created"}, w.outbox.subjects)
}

func TestCreateTheme_requiresThemedStorefrontChannel(t *testing.T) {
	w := newWorld(t)
	headless := themedChannel("chan-2")
	headless.StorefrontMode = channelsDomain.StorefrontModeHeadless
	w.channels.byID["chan-2"] = headless

	_, err := apptheme.NewCreateThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateThemeInput{
		SalesChannelID: "chan-2", Name: "X", PresetKey: theme.PresetDefault,
	})
	assert.ErrorIs(t, err, theme.ErrStorefrontNotEnabled)

	_, err = apptheme.NewCreateThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateThemeInput{
		SalesChannelID: "missing", Name: "X", PresetKey: theme.PresetDefault,
	})
	assert.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
	assert.Empty(t, w.themes.byID)
}

func TestCreateTheme_validatesNameAndPreset(t *testing.T) {
	w := newWorld(t)
	uc := apptheme.NewCreateThemeUseCase(w.deps)

	_, err := uc.Execute(tenantCtx(), apptheme.CreateThemeInput{SalesChannelID: "chan-1", Name: " ", PresetKey: theme.PresetDefault})
	assert.ErrorIs(t, err, theme.ErrInvalidThemeName)
	_, err = uc.Execute(tenantCtx(), apptheme.CreateThemeInput{SalesChannelID: "chan-1", Name: "X", PresetKey: "nope"})
	assert.ErrorIs(t, err, theme.ErrUnknownPreset)
}

func TestListThemes_paginatesPerChannel(t *testing.T) {
	w := newWorld(t)
	w.createTheme(t)
	w.createTheme(t)

	list, total, err := apptheme.NewListThemesUseCase(w.deps).Execute(tenantCtx(), "chan-1", 1, 1)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 1)
}

func TestGetTheme_includesDraftRevision(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	detail, err := apptheme.NewGetThemeUseCase(w.deps).Execute(tenantCtx(), th.ID)

	require.NoError(t, err)
	assert.Equal(t, th.ID, detail.Theme.ID)
	require.NotNil(t, detail.Draft)
	assert.Equal(t, 1, detail.Draft.Revision)
	assert.Nil(t, detail.Published)
}

func TestRenameTheme(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	got, err := apptheme.NewRenameThemeUseCase(w.deps).Execute(tenantCtx(), th.ID, "Winter")
	require.NoError(t, err)
	assert.Equal(t, "Winter", got.Name)
	assert.Equal(t, "Winter", w.themes.byID[th.ID].Name)

	_, err = apptheme.NewRenameThemeUseCase(w.deps).Execute(tenantCtx(), "missing", "Winter")
	assert.ErrorIs(t, err, theme.ErrThemeNotFound)
}

func TestDeleteTheme_refusesLiveTheme(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	w.publish(t, th.ID)

	err := apptheme.NewDeleteThemeUseCase(w.deps).Execute(tenantCtx(), th.ID)

	assert.ErrorIs(t, err, theme.ErrThemeIsLive)
	assert.Empty(t, w.themes.deleted)
}

func TestDeleteTheme_deletesAndRaisesEvent(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	require.NoError(t, apptheme.NewDeleteThemeUseCase(w.deps).Execute(tenantCtx(), th.ID))

	assert.Equal(t, []string{th.ID}, w.themes.deleted)
	assert.Contains(t, w.outbox.subjects, "storefront.theme.deleted")
}

func TestCreateTheme_startsFromThePresetsSettingsAndDocuments(t *testing.T) {
	w := newWorld(t)

	th, err := apptheme.NewCreateThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateThemeInput{
		SalesChannelID: "chan-1", Name: "Market", PresetKey: theme.PresetMercado, UserID: "user-1",
	})

	require.NoError(t, err)
	draft := w.draft(t, th.ID)
	want, _ := theme.PresetSettings(theme.PresetMercado)
	assert.Equal(t, want, draft.Settings)
	preset, _ := theme.PresetDocuments(theme.PresetMercado)
	assert.Len(t, w.docs.forVersion(draft.ID), len(preset))
}
