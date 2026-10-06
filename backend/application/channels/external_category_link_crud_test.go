package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListExternalCategoryLinks_returnsLinksForConnection(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{}
	linkRepo.listResult = []channelsDomain.ExternalCategoryLink{{ID: "link-1", ConnectionID: "conn-1", ExternalID: "20"}}
	uc := appchannels.NewListExternalCategoryLinksUseCase(newMockGormDB(t, 1), linkRepo)

	links, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, "20", links[0].ExternalID)
}

func TestUpsertExternalCategoryLink_success(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{}
	categoryRepo := &stubResolveCategoryRepo{byID: map[string]pmsDomain.Category{"cat-1": {ID: "cat-1", Slug: "beverages"}}}
	uc := appchannels.NewUpsertExternalCategoryLinkUseCase(newMockGormDB(t, 1), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	link, err := uc.Execute(context.Background(), appchannels.UpsertExternalCategoryLinkInput{
		ConnectionID: "conn-1", ExternalID: "20", CategoryID: "cat-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "conn-1", link.ConnectionID)
	assert.Equal(t, "20", link.ExternalID)
	assert.Equal(t, "cat-1", link.CategoryID)
	require.Len(t, linkRepo.upserted, 1)
}

func TestUpsertExternalCategoryLink_propagatesCategoryNotFound(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{}
	categoryRepo := &stubResolveCategoryRepo{}
	uc := appchannels.NewUpsertExternalCategoryLinkUseCase(newMockGormDB(t, 1), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpsertExternalCategoryLinkInput{
		ConnectionID: "conn-1", ExternalID: "20", CategoryID: "missing",
	})

	assert.ErrorIs(t, err, pmsDomain.ErrCategoryNotFound)
}

func TestUpsertExternalCategoryLink_propagatesRepoError(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{upsertErr: errors.New("db down")}
	categoryRepo := &stubResolveCategoryRepo{byID: map[string]pmsDomain.Category{"cat-1": {ID: "cat-1"}}}
	uc := appchannels.NewUpsertExternalCategoryLinkUseCase(newMockGormDB(t, 1), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpsertExternalCategoryLinkInput{
		ConnectionID: "conn-1", ExternalID: "20", CategoryID: "cat-1",
	})

	require.Error(t, err)
}

func TestDeleteExternalCategoryLink_success(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{}
	uc := appchannels.NewDeleteExternalCategoryLinkUseCase(newMockGormDB(t, 1), linkRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "link-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"link-1"}, linkRepo.deleted)
}

func TestDeleteExternalCategoryLink_propagatesRepoError(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{deleteErr: errors.New("db down")}
	uc := appchannels.NewDeleteExternalCategoryLinkUseCase(newMockGormDB(t, 1), linkRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "link-1")

	require.Error(t, err)
}
