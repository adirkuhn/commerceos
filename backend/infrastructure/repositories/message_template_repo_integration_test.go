//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestMessageTemplateRepo_Integration_CreateFindUpdateDeleteRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewMessageTemplateRepo(db)

	var created mcDomain.MessageTemplate
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = mcDomain.MessageTemplate{
			TenantID: tenant.IDFromContext(ctx), Name: "Order confirmed", Category: mcDomain.CategoryPurchase,
			Channels: []messaging.Channel{messaging.ChannelEmail, messaging.ChannelWhatsApp},
			Subject:  "Confirmed", Body: "Hi {{customer_name}}", Enabled: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *mcDomain.MessageTemplate
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, "Order confirmed", found.Name)
	assert.ElementsMatch(t, []messaging.Channel{messaging.ChannelEmail, messaging.ChannelWhatsApp}, found.Channels)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.Name = "Order confirmed v2"
		found.Enabled = false
		return repo.Update(ctx, found)
	})
	require.NoError(t, err)

	var listed []mcDomain.MessageTemplate
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = repo.ListByTenant(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "Order confirmed v2", listed[0].Name)
	assert.False(t, listed[0].Enabled)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Delete(ctx, created.ID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, created.ID)
		return err
	})
	assert.ErrorIs(t, err, mcDomain.ErrTemplateNotFound)
}
