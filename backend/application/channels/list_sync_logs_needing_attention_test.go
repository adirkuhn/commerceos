package channels_test

import (
	"context"
	"testing"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSyncLogsNeedingAttention_enrichesWithProductAndConnectionNames(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{needingAttention: []channelsDomain.SyncLogEntry{
		{ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1", Status: channelsDomain.SyncLogStatusError},
	}}
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"prod-1": {ID: "prod-1", Name: "Widget A"}}}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", StoreURL: "https://mystore.com"}}
	uc := channelsapp.NewListSyncLogsNeedingAttentionUseCase(newMockGormDB(t, 1), syncLogRepo, productRepo, connRepo)

	rows, total, err := uc.Execute(context.Background(), 1, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "Widget A", rows[0].ProductName)
	assert.Equal(t, "https://mystore.com", rows[0].StoreURL)
}

func TestListSyncLogsNeedingAttention_propagatesARepoError(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{needingAttnErr: assert.AnError}
	uc := channelsapp.NewListSyncLogsNeedingAttentionUseCase(newMockGormDB(t, 1), syncLogRepo, &stubChannelsProductRepo{}, &stubConnectionRepo{})

	_, _, err := uc.Execute(context.Background(), 1, 20)

	require.Error(t, err)
}
