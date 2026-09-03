package apikeys_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
)

func TestResolveAPIKey_returnsTenantIDForAMatchingHash(t *testing.T) {
	// Generate a real key + hash through CreateAPIKeyUseCase so this test
	// exercises the exact same hashing both use cases share, rather than
	// assuming they agree.
	repo := &stubAPIKeyRepo{resolveOut: "tenant-1"}
	createUC := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), repo)
	created, err := createUC.Execute(context.Background(), "Key A", "")
	require.NoError(t, err)

	resolveUC := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), repo)

	tenantID, err := resolveUC.Execute(context.Background(), created.PlaintextKey)

	require.NoError(t, err)
	require.Equal(t, "tenant-1", tenantID)
	require.Equal(t, repo.createdKey.KeyHash, repo.gotHash)
}

func TestResolveAPIKey_rejectsUnknownOrRevokedKey(t *testing.T) {
	repo := &stubAPIKeyRepo{resolveErr: apikeysDomain.ErrNotFound}
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), "wrong-key")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
}

func TestResolveAPIKey_rejectsEmptyKey(t *testing.T) {
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{})

	_, err := uc.Execute(context.Background(), "")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
}
