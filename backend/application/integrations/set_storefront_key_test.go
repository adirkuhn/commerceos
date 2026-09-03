package integrations_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
)

type stubStorefrontKeyHashSetter struct {
	gotHash string
	err     error
}

func (s *stubStorefrontKeyHashSetter) SetStorefrontKeyHash(_ context.Context, hash string) error {
	s.gotHash = hash
	return s.err
}

func TestSetStorefrontKey_generatesAndHashesKey(t *testing.T) {
	repo := &stubStorefrontKeyHashSetter{}
	uc := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.NotEmpty(t, result.PlaintextKey)

	sum := sha256.Sum256([]byte(result.PlaintextKey))
	require.Equal(t, hex.EncodeToString(sum[:]), repo.gotHash)
}

func TestSetStorefrontKey_rotatesToADifferentKeyOnEachCall(t *testing.T) {
	repo := &stubStorefrontKeyHashSetter{}
	uc := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)
	first, err := uc.Execute(context.Background())
	require.NoError(t, err)

	uc2 := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)
	second, err := uc2.Execute(context.Background())
	require.NoError(t, err)

	require.NotEqual(t, first.PlaintextKey, second.PlaintextKey)
}

func TestSetStorefrontKey_propagatesRepoError(t *testing.T) {
	repo := &stubStorefrontKeyHashSetter{err: errors.New("db down")}
	uc := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
