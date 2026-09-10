package theme_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appTheme "github.com/adirkuhn/commerceos/application/theme"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
)

func TestResolveEditSession_returnsTenantAndChannelForAValidToken(t *testing.T) {
	repo := &fakeEditSessionRepo{byToken: map[string]themeDomain.EditSession{
		"token-1": {TenantID: "tenant-1", SalesChannelID: "channel-1", Token: "token-1", ExpiresAt: time.Now().Add(time.Hour)},
	}}
	uc := appTheme.NewResolveEditSessionUseCase(newMockGormDB(t, 1), repo)

	resolved, err := uc.Execute(context.Background(), "token-1")

	require.NoError(t, err)
	require.Equal(t, "tenant-1", resolved.TenantID)
	require.Equal(t, "channel-1", resolved.SalesChannelID)
}

func TestResolveEditSession_rejectsUnknownToken(t *testing.T) {
	uc := appTheme.NewResolveEditSessionUseCase(newMockGormDB(t, 1), &fakeEditSessionRepo{})

	_, err := uc.Execute(context.Background(), "wrong-token")

	require.ErrorIs(t, err, themeDomain.ErrEditSessionNotFound)
}

func TestResolveEditSession_rejectsExpiredToken(t *testing.T) {
	repo := &fakeEditSessionRepo{byToken: map[string]themeDomain.EditSession{
		"token-1": {TenantID: "tenant-1", SalesChannelID: "channel-1", Token: "token-1", ExpiresAt: time.Now().Add(-time.Minute)},
	}}
	uc := appTheme.NewResolveEditSessionUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "token-1")

	require.ErrorIs(t, err, themeDomain.ErrEditSessionNotFound)
}

func TestResolveEditSession_rejectsEmptyToken(t *testing.T) {
	uc := appTheme.NewResolveEditSessionUseCase(newMockGormDB(t, 0), &fakeEditSessionRepo{})

	_, err := uc.Execute(context.Background(), "")

	require.ErrorIs(t, err, themeDomain.ErrEditSessionNotFound)
}
