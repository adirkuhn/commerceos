package apikeys_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type stubChannelRepo struct {
	byID   map[string]channelsDomain.SalesChannel
	gotIDs []string
}

func (s *stubChannelRepo) FindByID(_ context.Context, id string) (*channelsDomain.SalesChannel, error) {
	s.gotIDs = append(s.gotIDs, id)
	c, ok := s.byID[id]
	if !ok {
		return nil, channelsDomain.ErrSalesChannelNotFound
	}
	return &c, nil
}

func TestResolveAPIKey_returnsTenantIDForAMatchingHash(t *testing.T) {
	// Generate a real key + hash through CreateAPIKeyUseCase so this test
	// exercises the exact same hashing both use cases share, rather than
	// assuming they agree.
	repo := &stubAPIKeyRepo{resolveOut: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	createUC := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})
	created, err := createUC.Execute(context.Background(), "Key A", "", nil)
	require.NoError(t, err)

	resolveUC := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), repo, &stubChannelRepo{})

	resolved, err := resolveUC.Execute(context.Background(), created.PlaintextKey)

	require.NoError(t, err)
	require.Equal(t, "tenant-1", resolved.TenantID)
	require.Equal(t, repo.createdKey.KeyHash, repo.gotHash)
}

func TestResolveAPIKey_rejectsUnknownOrRevokedKey(t *testing.T) {
	repo := &stubAPIKeyRepo{resolveErr: apikeysDomain.ErrNotFound}
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), repo, &stubChannelRepo{})

	_, err := uc.Execute(context.Background(), "wrong-key")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
}

func TestResolveAPIKey_rejectsEmptyKey(t *testing.T) {
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{}, &stubChannelRepo{})

	_, err := uc.Execute(context.Background(), "")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
}

// TestResolveAPIKey_fallsBackToInternalSalesChannelID proves a bearer value
// that isn't a known hashed key, but is a UUID matching an internal
// SalesChannel's own id, resolves the same as a dedicated storefront key
// would -- what lets the admin app show that channel's id directly as its
// storefront URL (see frontend/src/pages/SalesChannels.tsx) instead of
// requiring a separately-managed key.
func TestResolveAPIKey_fallsBackToInternalSalesChannelID(t *testing.T) {
	channel := channelsDomain.SalesChannel{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", TenantID: "tenant-2", Type: channelsDomain.ChannelTypeInternal}
	channelRepo := &stubChannelRepo{byID: map[string]channelsDomain.SalesChannel{channel.ID: channel}}
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{resolveErr: apikeysDomain.ErrNotFound}, channelRepo)

	resolved, err := uc.Execute(context.Background(), channel.ID)

	require.NoError(t, err)
	require.Equal(t, "tenant-2", resolved.TenantID)
	require.NotNil(t, resolved.SalesChannelID)
	require.Equal(t, channel.ID, *resolved.SalesChannelID)
}

// TestResolveAPIKey_rejectsNonInternalSalesChannelID proves a channel id
// belonging to a non-Internal channel type (WooCommerce/Bol/WhatsApp) --
// none of which are meant to grant public/anonymous-shopper access -- is
// rejected exactly like an unknown key, not silently resolved.
func TestResolveAPIKey_rejectsNonInternalSalesChannelID(t *testing.T) {
	channel := channelsDomain.SalesChannel{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", TenantID: "tenant-2", Type: channelsDomain.ChannelTypeWooCommerce}
	channelRepo := &stubChannelRepo{byID: map[string]channelsDomain.SalesChannel{channel.ID: channel}}
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{resolveErr: apikeysDomain.ErrNotFound}, channelRepo)

	_, err := uc.Execute(context.Background(), channel.ID)

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
}

// TestResolveAPIKey_nonUUIDBearerNeverQueriesChannelRepo proves an
// ordinary (non-UUID-shaped) unknown key never even attempts the channel
// fallback lookup -- guards against a malformed-UUID query error replacing
// the real ErrNotFound for the overwhelmingly common "just a wrong key"
// case.
func TestResolveAPIKey_nonUUIDBearerNeverQueriesChannelRepo(t *testing.T) {
	channelRepo := &stubChannelRepo{}
	uc := appapikeys.NewResolveAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{resolveErr: apikeysDomain.ErrNotFound}, channelRepo)

	_, err := uc.Execute(context.Background(), "not-a-uuid-at-all")

	require.Error(t, err)
	require.True(t, errors.Is(err, apikeysDomain.ErrNotFound))
	require.Empty(t, channelRepo.gotIDs)
}
