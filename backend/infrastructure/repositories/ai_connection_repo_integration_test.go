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

func aiConn(tenantID string, provider integrationsDomain.AIProvider, position int, label string) *integrationsDomain.AIConnection {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &integrationsDomain.AIConnection{
		TenantID: tenantID, Provider: provider, APIKeyEncrypted: "enc", Position: position, Label: label, Enabled: true,
		Model: "m", FallbackModels: "f", Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
	}
}

func TestAIConnectionRepo_severalPerTenantListedInRoutingOrder(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAIConnectionRepo(db)
	tenantID, ctx := newTestTenant()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		second := aiConn(tenantID, integrationsDomain.AIProviderNVIDIA, 1, "NVIDIA key 2")
		first := aiConn(tenantID, integrationsDomain.AIProviderNVIDIA, 0, "NVIDIA key 1")
		third := aiConn(tenantID, integrationsDomain.AIProviderGemini, 2, "")
		third.Enabled = false
		for _, c := range []*integrationsDomain.AIConnection{second, first, third} {
			require.NoError(t, repo.Create(ctx, c))
			require.NotEmpty(t, c.ID)
		}

		list, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 3, "the same provider can be connected twice")
		assert.Equal(t, []string{"NVIDIA key 1", "NVIDIA key 2", ""}, []string{list[0].Label, list[1].Label, list[2].Label})
		assert.False(t, list[2].Enabled)
		assert.Equal(t, "f", list[0].FallbackModels)

		got, err := repo.Get(ctx, second.ID)
		require.NoError(t, err)
		got.Position, got.Label, got.Enabled, got.Model = 0, "renamed", false, "m2"
		require.NoError(t, repo.Update(ctx, got))
		again, err := repo.Get(ctx, second.ID)
		require.NoError(t, err)
		assert.Equal(t, "renamed", again.Label)
		assert.Equal(t, "m2", again.Model)
		assert.False(t, again.Enabled)
		assert.Equal(t, 0, again.Position)

		require.NoError(t, repo.Delete(ctx, first.ID))
		_, err = repo.Get(ctx, first.ID)
		assert.ErrorIs(t, err, integrationsDomain.ErrAIConnectionNotFound)
		assert.ErrorIs(t, repo.Delete(ctx, first.ID), integrationsDomain.ErrAIConnectionNotFound)
		assert.ErrorIs(t, repo.Update(ctx, first), integrationsDomain.ErrAIConnectionNotFound)
		return nil
	})
	require.NoError(t, err)
}

func TestAIConnectionRepo_tenantsCannotSeeEachOther(t *testing.T) {
	super, dsn := newIntegrationDBAndDSN(t)
	app := newAppRoleDB(t, dsn)
	tenantA, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	c := aiConn(tenantA, integrationsDomain.AIProviderOpenAI, 0, "")
	require.NoError(t, tenant.WithTenant(ctxA, super, func(ctx context.Context) error {
		return repositories.NewAIConnectionRepo(super).Create(ctx, c)
	}))

	err := tenant.WithTenant(ctxB, app, func(ctx context.Context) error {
		list, err := repositories.NewAIConnectionRepo(app).List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
		_, err = repositories.NewAIConnectionRepo(app).Get(ctx, c.ID)
		assert.ErrorIs(t, err, integrationsDomain.ErrAIConnectionNotFound)
		return nil
	})
	require.NoError(t, err)
}
