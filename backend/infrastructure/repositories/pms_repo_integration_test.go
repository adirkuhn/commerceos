//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestProductRepo_Integration_SearchIsAccentInsensitive proves the
// unaccent()-based search (022_unaccent_extension.sql) actually works
// against real Postgres -- sqlmock-based tests can only assert the SQL
// string sent, not that unaccent() as a function actually exists and
// behaves as expected in the target database.
func TestProductRepo_Integration_SearchIsAccentInsensitive(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for _, name := range []string{"Açaí na tigela", "Feijão preto", "Pão de queijo"} {
			p := &pmsDomain.Product{
				TenantID: tenant.IDFromContext(ctx), Name: name, Slug: name,
				Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := productRepo.Create(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		query string
		want  string
	}{
		{"acai", "Açaí na tigela"},
		{"AÇAÍ", "Açaí na tigela"},
		{"feijao", "Feijão preto"},
		{"pao", "Pão de queijo"},
	} {
		var products []pmsDomain.Product
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			search := tc.query
			products, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search})
			return err
		})
		require.NoError(t, err)
		require.Len(t, products, 1, "query %q should match exactly one product", tc.query)
		require.Equal(t, tc.want, products[0].Name, "query %q", tc.query)
	}
}
