//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestProductRepo_Integration_ListLinksHonoursTheFilterAndLimit(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewProductRepo(db)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	var excludedID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for i, p := range []struct {
			slug   string
			status pmsDomain.ProductStatus
		}{{"old", pmsDomain.ProductStatusActive}, {"new", pmsDomain.ProductStatusActive}, {"draft", pmsDomain.ProductStatusDraft}, {"hidden", pmsDomain.ProductStatusActive}} {
			at := base.Add(time.Duration(i) * time.Hour)
			prod := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: p.slug, Slug: p.slug, Status: p.status, CreatedAt: at, UpdatedAt: at}
			if err := repo.Create(ctx, prod); err != nil {
				return err
			}
			if p.slug == "hidden" {
				excludedID = prod.ID
			}
		}
		return nil
	})
	require.NoError(t, err)

	active := pmsDomain.ProductStatusActive
	var links, limited []pmsDomain.ProductLink
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if links, err = repo.ListLinks(ctx, pmsDomain.ProductFilter{Status: &active, ExcludeIDs: []string{excludedID}, Limit: 100}); err != nil {
			return err
		}
		limited, err = repo.ListLinks(ctx, pmsDomain.ProductFilter{Status: &active, Limit: 1})
		return err
	})
	require.NoError(t, err)
	require.Len(t, links, 2)
	assert.Equal(t, "new", links[0].Slug)
	assert.Equal(t, "old", links[1].Slug)
	assert.True(t, links[0].UpdatedAt.Equal(base.Add(time.Hour)))
	assert.Len(t, limited, 1)
}
