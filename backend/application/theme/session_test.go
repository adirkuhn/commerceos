package theme_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestCreateSession_returnsRawTokenAndStoresHash(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	res, err := apptheme.NewCreateSessionUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateSessionInput{
		ThemeID: th.ID, UserID: "user-1", Role: "customer_admin", Purpose: theme.PurposeEdit,
	})

	require.NoError(t, err)
	assert.Equal(t, "token-1", res.Token)
	stored, ok := w.sessions.byHash[theme.HashToken("token-1")]
	require.True(t, ok)
	assert.Equal(t, th.ID, stored.ThemeID)
	assert.Equal(t, stored.ExpiresAt, res.ExpiresAt)
}

func TestCreateSession_unknownThemeOrHeadlessChannel(t *testing.T) {
	w := newWorld(t)
	_, err := apptheme.NewCreateSessionUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateSessionInput{ThemeID: "missing", Purpose: theme.PurposeEdit})
	assert.ErrorIs(t, err, theme.ErrThemeNotFound)

	th := w.createTheme(t)
	ch := w.channels.byID["chan-1"]
	ch.StorefrontMode = "headless"
	w.channels.byID["chan-1"] = ch
	_, err = apptheme.NewCreateSessionUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateSessionInput{ThemeID: th.ID, Purpose: theme.PurposeEdit})
	assert.ErrorIs(t, err, theme.ErrStorefrontNotEnabled)
}

func TestResolveSession(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	_, err := apptheme.NewCreateSessionUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateSessionInput{
		ThemeID: th.ID, UserID: "user-1", Role: "customer_admin", Purpose: theme.PurposeEdit,
	})
	require.NoError(t, err)
	uc := apptheme.NewResolveSessionUseCase(w.deps)

	s, err := uc.Execute(context.Background(), "token-1", theme.PurposeEdit)
	require.NoError(t, err)
	assert.Equal(t, th.ID, s.ThemeID)
	assert.Equal(t, "tenant-1", s.TenantID)

	_, err = uc.Execute(context.Background(), "token-1", theme.PurposePreview)
	assert.ErrorIs(t, err, theme.ErrSessionInvalid, "an edit token is not a preview token")
	_, err = uc.Execute(context.Background(), "unknown", theme.PurposeEdit)
	assert.ErrorIs(t, err, theme.ErrSessionInvalid)
}
