//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestAIConnectionRepo_UpsertNVIDIAThenGet(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAIConnectionRepo(db)
	tenantID, ctx := newTestTenant()
	now := time.Now().UTC().Truncate(time.Microsecond)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		require.NoError(t, repo.Upsert(ctx, &integrationsDomain.AIConnection{
			TenantID: tenantID, Provider: integrationsDomain.AIProviderNVIDIA, APIKeyEncrypted: "enc",
			Model: "meta/llama-3.3-70b-instruct", FallbackModels: "meta/llama-3.1-8b-instruct",
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}))
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, integrationsDomain.AIProviderNVIDIA, got.Provider)
		assert.Equal(t, "meta/llama-3.1-8b-instruct", got.FallbackModels)
		return nil
	})
	require.NoError(t, err)
}
