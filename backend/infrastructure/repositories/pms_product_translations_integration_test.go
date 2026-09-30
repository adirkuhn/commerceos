//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestProductRepo_Integration_TranslationsRoundTripSurviveUpdatesAndAreSearched(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewProductRepo(db)
	var p pmsDomain.Product

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		p = pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Pão de queijo", Slug: "pao-de-queijo",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := repo.Create(ctx, &p); err != nil {
			return err
		}
		return repo.SetTranslations(ctx, p.ID, map[locale.Locale]pmsDomain.ProductTexts{
			locale.English: {Name: "Cheese bread", Description: "Warm and chewy"},
		})
	}))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "Cheese bread", found.Translations[locale.English].Name)

		found.Name = "Pão de queijo mineiro"
		require.NoError(t, repo.Update(ctx, found))
		again, err := repo.FindByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "Warm and chewy", again.Translations[locale.English].Description, "a product update leaves translations alone")

		search := "cheese"
		list, _, err := repo.List(ctx, pmsDomain.ProductFilter{Search: &search})
		require.NoError(t, err)
		require.Len(t, list, 1, "shoppers searching in English find it")
		assert.Equal(t, p.ID, list[0].ID)
		return nil
	}))

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.SetTranslations(ctx, "00000000-0000-0000-0000-000000000000", nil)
	})
	assert.ErrorIs(t, err, pmsDomain.ErrProductNotFound)
}
