package integrations_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubStorefrontKeyResolver struct {
	tenantID string
	err      error
	gotHash  string
}

func (s *stubStorefrontKeyResolver) FindTenantIDByStorefrontKeyHash(_ context.Context, hash string) (string, error) {
	s.gotHash = hash
	if s.err != nil {
		return "", s.err
	}
	return s.tenantID, nil
}

func TestResolveStorefrontKey_returnsTenantIDForAMatchingHash(t *testing.T) {
	repo := &stubStorefrontKeyResolver{tenantID: "tenant-1"}
	uc := appintegrations.NewResolveStorefrontKeyUseCase(newMockGormDB(t), repo)

	tenantID, err := uc.Execute(context.Background(), "some-plaintext-key")

	require.NoError(t, err)
	require.Equal(t, "tenant-1", tenantID)
	sum := sha256.Sum256([]byte("some-plaintext-key"))
	require.Equal(t, hex.EncodeToString(sum[:]), repo.gotHash)
}

func TestResolveStorefrontKey_rejectsUnknownKey(t *testing.T) {
	repo := &stubStorefrontKeyResolver{err: settingsDomain.ErrStorefrontKeyNotFound}
	uc := appintegrations.NewResolveStorefrontKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), "wrong-key")

	require.Error(t, err)
	require.True(t, errors.Is(err, settingsDomain.ErrStorefrontKeyNotFound))
}

func TestResolveStorefrontKey_rejectsEmptyKey(t *testing.T) {
	uc := appintegrations.NewResolveStorefrontKeyUseCase(newMockGormDB(t), &stubStorefrontKeyResolver{})

	_, err := uc.Execute(context.Background(), "")

	require.Error(t, err)
	require.True(t, errors.Is(err, settingsDomain.ErrStorefrontKeyNotFound))
}
