package apikeys_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubAPIKeyRepo struct {
	createErr  error
	createdKey *apikeysDomain.APIKey
	listOut    []apikeysDomain.APIKey
	listErr    error
	revokeErr  error
	revokedID  string
	resolveOut string
	resolveErr error
	gotHash    string
}

func (s *stubAPIKeyRepo) Create(_ context.Context, k *apikeysDomain.APIKey) error {
	if s.createErr != nil {
		return s.createErr
	}
	k.ID = "key-1"
	s.createdKey = k
	return nil
}

func (s *stubAPIKeyRepo) ListByTenant(_ context.Context) ([]apikeysDomain.APIKey, error) {
	return s.listOut, s.listErr
}

func (s *stubAPIKeyRepo) Revoke(_ context.Context, id string) error {
	s.revokedID = id
	return s.revokeErr
}

func (s *stubAPIKeyRepo) FindActiveTenantIDByHash(_ context.Context, hash string) (string, error) {
	s.gotHash = hash
	if s.resolveErr != nil {
		return "", s.resolveErr
	}
	return s.resolveOut, nil
}

func TestCreateAPIKey_generatesAndHashesKey(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	uc := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), repo)
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")

	result, err := uc.Execute(ctx, "My storefront", "WooCommerce prod")

	require.NoError(t, err)
	require.Equal(t, "key-1", result.ID)
	require.Equal(t, "My storefront", result.Name)
	require.Equal(t, "WooCommerce prod", result.Description)
	require.NotEmpty(t, result.PlaintextKey)
	require.NotEqual(t, result.PlaintextKey, repo.createdKey.KeyHash)
	require.NotEmpty(t, repo.createdKey.KeyHash)
	// Regression check: Create must be called with the tenant ID set --
	// this was missing once already, and the DB rejected an empty
	// tenant_id ("" is not a valid uuid), a failure sqlmock can't catch.
	require.Equal(t, "tenant-1", repo.createdKey.TenantID)
}

func TestCreateAPIKey_rejectsEmptyName(t *testing.T) {
	uc := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{})

	_, err := uc.Execute(context.Background(), "", "")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNameRequired))
}

func TestCreateAPIKey_generatesADifferentKeyEachCall(t *testing.T) {
	repo := &stubAPIKeyRepo{}
	uc := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), repo)
	first, err := uc.Execute(context.Background(), "Key A", "")
	require.NoError(t, err)

	uc2 := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), repo)
	second, err := uc2.Execute(context.Background(), "Key B", "")
	require.NoError(t, err)

	require.NotEqual(t, first.PlaintextKey, second.PlaintextKey)
}

func TestCreateAPIKey_propagatesRepoError(t *testing.T) {
	repo := &stubAPIKeyRepo{createErr: errors.New("db down")}
	uc := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), "Key A", "")

	require.Error(t, err)
}
