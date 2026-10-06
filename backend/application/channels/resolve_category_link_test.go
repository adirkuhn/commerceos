package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// stubResolveCategoryLinkRepo is local to this file -- the package's
// existing stubCategoryRepo (see testutil_test.go) hardcodes FindByID to
// always report "not found", which can't exercise this use case's slug
// -match path (it needs the category's real slug to compare against Woo's
// list).

type stubExternalCategoryLinkRepo struct {
	byConnAndCategory map[string]channelsDomain.ExternalCategoryLink // key: connectionID+"|"+categoryID
	upserted          []channelsDomain.ExternalCategoryLink
	upsertErr         error
	listResult        []channelsDomain.ExternalCategoryLink
	deleted           []string
	deleteErr         error
}

func (s *stubExternalCategoryLinkRepo) Upsert(_ context.Context, l *channelsDomain.ExternalCategoryLink) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	l.ID = "link-1"
	s.upserted = append(s.upserted, *l)
	return nil
}
func (s *stubExternalCategoryLinkRepo) FindByConnectionAndExternalID(_ context.Context, _, _ string) (*channelsDomain.ExternalCategoryLink, error) {
	return nil, channelsDomain.ErrExternalCategoryLinkNotFound
}
func (s *stubExternalCategoryLinkRepo) FindByConnectionAndCategory(_ context.Context, connectionID, categoryID string) (*channelsDomain.ExternalCategoryLink, error) {
	if l, ok := s.byConnAndCategory[connectionID+"|"+categoryID]; ok {
		return &l, nil
	}
	return nil, channelsDomain.ErrExternalCategoryLinkNotFound
}
func (s *stubExternalCategoryLinkRepo) ListByConnection(_ context.Context, _ string) ([]channelsDomain.ExternalCategoryLink, error) {
	return s.listResult, nil
}
func (s *stubExternalCategoryLinkRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubResolveCategoryRepo struct {
	pmsDomain.CategoryRepository
	byID map[string]pmsDomain.Category
}

func (s *stubResolveCategoryRepo) FindByID(_ context.Context, id string) (*pmsDomain.Category, error) {
	if c, ok := s.byID[id]; ok {
		return &c, nil
	}
	return nil, pmsDomain.ErrCategoryNotFound
}

func TestResolveCategoryLink_returnsExistingLinkWithoutCallingWoo(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{byConnAndCategory: map[string]channelsDomain.ExternalCategoryLink{
		"conn-1|cat-1": {ID: "link-1", ConnectionID: "conn-1", CategoryID: "cat-1", ExternalID: "900"},
	}}
	categoryRepo := &stubResolveCategoryRepo{}
	wooClient := &stubWooClient{categoriesErr: errors.New("must not be called")}
	uc := appchannels.NewResolveCategoryLinkUseCase(newMockGormDB(t, 1), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	externalID, err := uc.Execute(context.Background(), wooClient, "conn-1", "cat-1")

	require.NoError(t, err)
	require.Equal(t, "900", externalID)
}

func TestResolveCategoryLink_matchesBySlugAndPersistsTheLink(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{byConnAndCategory: map[string]channelsDomain.ExternalCategoryLink{}}
	categoryRepo := &stubResolveCategoryRepo{byID: map[string]pmsDomain.Category{
		"cat-1": {ID: "cat-1", Slug: "beverages"},
	}}
	wooClient := &stubWooClient{categories: []integrationsDomain.WooCategory{
		{ExternalID: "10", Name: "Snacks", Slug: "snacks"},
		{ExternalID: "20", Name: "Beverages", Slug: "beverages"},
	}}
	uc := appchannels.NewResolveCategoryLinkUseCase(newMockGormDB(t, 3), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	externalID, err := uc.Execute(context.Background(), wooClient, "conn-1", "cat-1")

	require.NoError(t, err)
	require.Equal(t, "20", externalID)
	require.Len(t, linkRepo.upserted, 1)
	require.Equal(t, "conn-1", linkRepo.upserted[0].ConnectionID)
	require.Equal(t, "cat-1", linkRepo.upserted[0].CategoryID)
	require.Equal(t, "20", linkRepo.upserted[0].ExternalID)
}

func TestResolveCategoryLink_noSlugMatchReturnsEmptyWithoutError(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{byConnAndCategory: map[string]channelsDomain.ExternalCategoryLink{}}
	categoryRepo := &stubResolveCategoryRepo{byID: map[string]pmsDomain.Category{
		"cat-1": {ID: "cat-1", Slug: "beverages"},
	}}
	wooClient := &stubWooClient{categories: []integrationsDomain.WooCategory{
		{ExternalID: "10", Name: "Snacks", Slug: "snacks"},
	}}
	uc := appchannels.NewResolveCategoryLinkUseCase(newMockGormDB(t, 2), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	externalID, err := uc.Execute(context.Background(), wooClient, "conn-1", "cat-1")

	require.NoError(t, err)
	require.Equal(t, "", externalID)
	require.Empty(t, linkRepo.upserted, "no match must not persist a link")
}

func TestResolveCategoryLink_unreachableStoreDegradesToEmptyInsteadOfFailing(t *testing.T) {
	linkRepo := &stubExternalCategoryLinkRepo{byConnAndCategory: map[string]channelsDomain.ExternalCategoryLink{}}
	categoryRepo := &stubResolveCategoryRepo{byID: map[string]pmsDomain.Category{
		"cat-1": {ID: "cat-1", Slug: "beverages"},
	}}
	wooClient := &stubWooClient{categoriesErr: errors.New("connection refused")}
	uc := appchannels.NewResolveCategoryLinkUseCase(newMockGormDB(t, 2), linkRepo, categoryRepo, &eventrecordtest.Outbox{})

	externalID, err := uc.Execute(context.Background(), wooClient, "conn-1", "cat-1")

	require.NoError(t, err)
	require.Equal(t, "", externalID)
}
