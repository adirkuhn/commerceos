package apikeys_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
)

func TestListAPIKeys_returnsRepoResult(t *testing.T) {
	repo := &stubAPIKeyRepo{listOut: []apikeysDomain.APIKey{
		{ID: "key-1", Name: "Key A"},
		{ID: "key-2", Name: "Key B"},
	}}
	uc := appapikeys.NewListAPIKeysUseCase(newMockGormDB(t), repo)

	keys, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.Equal(t, "Key A", keys[0].Name)
}

func TestListAPIKeys_propagatesRepoError(t *testing.T) {
	repo := &stubAPIKeyRepo{listErr: errors.New("db down")}
	uc := appapikeys.NewListAPIKeysUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
