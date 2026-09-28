package storefront

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/collections"
)

// StorefrontCollection is a published collection as shoppers see it.
type StorefrontCollection struct {
	collections.Collection
	ImageURL *string
}

type publishedCollectionGetter interface {
	Execute(ctx context.Context, slug string) (*collections.Collection, error)
}

type collectionLister interface {
	Execute(ctx context.Context, publishedOnly bool, page, perPage int) ([]collections.Collection, int64, error)
}

type assetURLResolver interface {
	Execute(ctx context.Context, ids []string) (map[string]string, error)
}

func withImages(ctx context.Context, assets assetURLResolver, list []collections.Collection) ([]StorefrontCollection, error) {
	var ids []string
	for _, c := range list {
		if c.ImageAssetID != nil {
			ids = append(ids, *c.ImageAssetID)
		}
	}
	urls, err := assets.Execute(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]StorefrontCollection, len(list))
	for i, c := range list {
		out[i] = StorefrontCollection{Collection: c}
		if c.ImageAssetID != nil {
			if u, ok := urls[*c.ImageAssetID]; ok {
				out[i].ImageURL = &u
			}
		}
	}
	return out, nil
}

// GetStorefrontCollectionUseCase: one published collection by slug.
type GetStorefrontCollectionUseCase struct {
	get    publishedCollectionGetter
	assets assetURLResolver
}

func NewGetStorefrontCollectionUseCase(get publishedCollectionGetter, assets assetURLResolver) *GetStorefrontCollectionUseCase {
	return &GetStorefrontCollectionUseCase{get: get, assets: assets}
}

func (uc *GetStorefrontCollectionUseCase) Execute(ctx context.Context, slug string) (*StorefrontCollection, error) {
	c, err := uc.get.Execute(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("getStorefrontCollection: %w", err)
	}
	out, err := withImages(ctx, uc.assets, []collections.Collection{*c})
	if err != nil {
		return nil, fmt.Errorf("getStorefrontCollection: %w", err)
	}
	return &out[0], nil
}

// ListStorefrontCollectionsUseCase: the published collections, by name.
type ListStorefrontCollectionsUseCase struct {
	list   collectionLister
	assets assetURLResolver
}

func NewListStorefrontCollectionsUseCase(list collectionLister, assets assetURLResolver) *ListStorefrontCollectionsUseCase {
	return &ListStorefrontCollectionsUseCase{list: list, assets: assets}
}

func (uc *ListStorefrontCollectionsUseCase) Execute(ctx context.Context, page, perPage int) ([]StorefrontCollection, int64, error) {
	list, total, err := uc.list.Execute(ctx, true, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("listStorefrontCollections: %w", err)
	}
	out, err := withImages(ctx, uc.assets, list)
	if err != nil {
		return nil, 0, fmt.Errorf("listStorefrontCollections: %w", err)
	}
	return out, total, nil
}
