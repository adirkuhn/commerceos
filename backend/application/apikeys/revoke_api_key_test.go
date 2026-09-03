package apikeys_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
)

func TestRevokeAPIKey_callsRepoWithID(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	uc := appapikeys.NewRevokeAPIKeyUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "key-1")

	require.NoError(t, err)
	require.Equal(t, "key-1", repo.revokedID)
}

func TestRevokeAPIKey_propagatesNotFound(t *testing.T) {
	repo := &stubAPIKeyRepo{revokeErr: apikeysDomain.ErrNotFound}
	uc := appapikeys.NewRevokeAPIKeyUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
}
