package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/domain/collections"
)

type stubPublished struct{ c *collections.Collection }

func (s stubPublished) Execute(context.Context, string) (*collections.Collection, error) {
	if s.c == nil {
		return nil, collections.ErrCollectionNotFound
	}
	return s.c, nil
}

type stubPublishedList struct {
	list      []collections.Collection
	published bool
}

func (s *stubPublishedList) Execute(_ context.Context, published bool, _, _ int) ([]collections.Collection, int64, error) {
	s.published = published
	return s.list, int64(len(s.list)), nil
}

type stubAssetURLs map[string]string

func (s stubAssetURLs) Execute(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if u, ok := s[id]; ok {
			out[id] = u
		}
	}
	return out, nil
}

func col(id string, image *string) collections.Collection {
	c := collections.Collection{ID: id}
	c.Name, c.Slug, c.ImageAssetID = id, id, image
	return c
}

func TestGetStorefrontCollection_resolvesTheImage(t *testing.T) {
	img := "asset-1"
	c := col("summer", &img)
	uc := appstorefront.NewGetStorefrontCollectionUseCase(stubPublished{c: &c}, stubAssetURLs{"asset-1": "https://cdn/s.jpg"})

	got, err := uc.Execute(context.Background(), "summer")
	require.NoError(t, err)
	require.NotNil(t, got.ImageURL)
	assert.Equal(t, "https://cdn/s.jpg", *got.ImageURL)

	_, err = appstorefront.NewGetStorefrontCollectionUseCase(stubPublished{}, stubAssetURLs{}).Execute(context.Background(), "x")
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func TestListStorefrontCollections_publishedOnlyWithImages(t *testing.T) {
	img := "asset-1"
	list := &stubPublishedList{list: []collections.Collection{col("a", &img), col("b", nil)}}
	uc := appstorefront.NewListStorefrontCollectionsUseCase(list, stubAssetURLs{"asset-1": "https://cdn/a.jpg"})

	got, total, err := uc.Execute(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.True(t, list.published)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "https://cdn/a.jpg", *got[0].ImageURL)
	assert.Nil(t, got[1].ImageURL)
}
