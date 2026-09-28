package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveAssetURLsUseCase maps media-library asset ids to their public
// URLs -- for other contexts (e.g. a collection's image) that refer to
// theme assets by id. Unknown ids are left out.
type ResolveAssetURLsUseCase struct{ deps Deps }

func NewResolveAssetURLsUseCase(deps Deps) *ResolveAssetURLsUseCase { return &ResolveAssetURLsUseCase{deps: deps} }

func (uc *ResolveAssetURLsUseCase) Execute(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		assets, err := uc.deps.resolveAssets(ctx, ids)
		for id, a := range assets {
			out[id] = a.URL
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("resolveAssetURLs: %w", err)
	}
	return out, nil
}
