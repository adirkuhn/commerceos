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

func TestAutomationRuleRepo_Integration_CreateFindUpdateDeleteRoundTrip(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	templateRepo := repositories.NewMessageTemplateRepo(db)
	repo := repositories.NewAutomationRuleRepo(db)

	var template mcDomain.MessageTemplate
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		template = mcDomain.MessageTemplate{
			TenantID: tenant.IDFromContext(ctx), Name: "Left at collect point", Category: mcDomain.CategoryTracking,
			Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Body: "Your package is at a collection point",
			Enabled: true, CreatedAt: now, UpdatedAt: now,
		}
		return templateRepo.Create(ctx, &template)
	})
	require.NoError(t, err)

	var created mcDomain.AutomationRule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		override := messaging.ChannelWhatsApp
		created = mcDomain.AutomationRule{
			TenantID: tenant.IDFromContext(ctx), Name: "Collect point notice", TriggerSubject: "shipping.shipment.fulfillment_signal",
			Conditions:      []mcDomain.Condition{{Field: "status_code", Operator: mcDomain.OperatorIn, Value: []any{float64(8), float64(19)}}},
			TemplateID:      template.ID,
			ChannelOverride: &override,
			Enabled:         true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *mcDomain.AutomationRule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, "Collect point notice", found.Name)
	require.Len(t, found.Conditions, 1)
	assert.Equal(t, "status_code", found.Conditions[0].Field)
	assert.Equal(t, mcDomain.OperatorIn, found.Conditions[0].Operator)
	require.NotNil(t, found.ChannelOverride)
	assert.Equal(t, messaging.ChannelWhatsApp, *found.ChannelOverride)

	var bySubject []mcDomain.AutomationRule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		bySubject, err = repo.ListActiveBySubject(ctx, "shipping.shipment.fulfillment_signal")
		return err
	})
	require.NoError(t, err)
	require.Len(t, bySubject, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.Enabled = false
		return repo.Update(ctx, found)
	})
	require.NoError(t, err)

	var afterDisable []mcDomain.AutomationRule
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		afterDisable, err = repo.ListActiveBySubject(ctx, "shipping.shipment.fulfillment_signal")
		return err
	})
	require.NoError(t, err)
	assert.Len(t, afterDisable, 0)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Delete(ctx, created.ID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, created.ID)
		return err
	})
	assert.ErrorIs(t, err, mcDomain.ErrRuleNotFound)
}
