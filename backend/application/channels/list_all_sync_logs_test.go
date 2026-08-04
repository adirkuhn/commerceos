package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListAllSyncLogs_enrichesEntriesWithProductNameAndStoreURL(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{all: []channelsDomain.SyncLogEntry{
		{ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1", Status: channelsDomain.SyncLogStatusSuccess},
	}}
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"prod-1": {ID: "prod-1", Name: "T-Shirt"}}}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", StoreURL: "https://mystore.com"}}
	uc := appchannels.NewListAllSyncLogsUseCase(newMockGormDB(t, 1), syncLogRepo, productRepo, connRepo)

	entries, total, err := uc.Execute(context.Background(), 1, 20)

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, entries, 1)
	require.Equal(t, "T-Shirt", entries[0].ProductName)
	require.Equal(t, "https://mystore.com", entries[0].StoreURL)
}

func TestListAllSyncLogs_degradesGracefullyWhenProductWasDeleted(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{all: []channelsDomain.SyncLogEntry{
		{ID: "log-1", ProductID: "gone", ConnectionID: "conn-1", Status: channelsDomain.SyncLogStatusSuccess},
	}}
	productRepo := &stubChannelsProductRepo{}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", StoreURL: "https://mystore.com"}}
	uc := appchannels.NewListAllSyncLogsUseCase(newMockGormDB(t, 1), syncLogRepo, productRepo, connRepo)

	entries, _, err := uc.Execute(context.Background(), 1, 20)

	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Empty(t, entries[0].ProductName)
}

func TestListAllSyncLogs_paginates(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{all: []channelsDomain.SyncLogEntry{
		{ID: "log-2", ProductID: "prod-1", ConnectionID: "conn-1"},
		{ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1"},
	}}
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"prod-1": {ID: "prod-1", Name: "T-Shirt"}}}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", StoreURL: "https://mystore.com"}}
	uc := appchannels.NewListAllSyncLogsUseCase(newMockGormDB(t, 1), syncLogRepo, productRepo, connRepo)

	entries, total, err := uc.Execute(context.Background(), 2, 1)

	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, entries, 1)
	require.Equal(t, "log-1", entries[0].ID)
}
