//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestNewsletterRepo_Integration_lifecycleSearchEraseAndRLS(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	_, ctx := newTestTenant()
	_, other := newTestTenant()
	repo := repositories.NewNewsletterRepo(db)
	now := time.Now()

	var ann newsletter.Subscriber
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		ann = newsletter.NewSubscriber(tenant.IDFromContext(ctx), "", "ann@example.com", "c1", "u1", now)
		if err := repo.Create(ctx, &ann); err != nil {
			return err
		}
		bob := newsletter.NewSubscriber(tenant.IDFromContext(ctx), "", "bob_1@example.com", "c2", "u2", now)
		return repo.Create(ctx, &bob)
	}))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found, err := repo.FindByConfirmTokenHash(ctx, newsletter.HashToken("c1"))
		require.NoError(t, err)
		require.NotNil(t, found)
		require.NoError(t, found.Confirm(now))
		require.NoError(t, repo.Update(ctx, found))

		again, err := repo.FindByConfirmTokenHash(ctx, newsletter.HashToken("c1"))
		require.NoError(t, err)
		assert.Nil(t, again, "confirm token cleared after use")

		byUnsub, err := repo.FindByUnsubscribeToken(ctx, "u1")
		require.NoError(t, err)
		assert.Equal(t, newsletter.StatusConfirmed, byUnsub.Status)

		confirmed := newsletter.StatusConfirmed
		list, total, err := repo.List(ctx, newsletter.ListFilter{Status: &confirmed, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "ann@example.com", list[0].Email)

		_, total, err = repo.List(ctx, newsletter.ListFilter{Search: "_", Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total, "search is literal, not a LIKE pattern")
		return nil
	}))

	require.NoError(t, tenant.WithTenant(other, db, func(ctx context.Context) error {
		s, err := repo.FindByEmail(ctx, "ann@example.com")
		require.NoError(t, err)
		assert.Nil(t, s, "RLS hides other tenants' subscribers")
		assert.ErrorIs(t, repo.Delete(ctx, ann.ID), newsletter.ErrSubscriberNotFound)
		dup := newsletter.NewSubscriber(tenant.IDFromContext(ctx), "", "ann@example.com", "c9", "u9", now)
		return repo.Create(ctx, &dup)
	}), "the same address can subscribe to another tenant's stores")

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		require.NoError(t, repo.Delete(ctx, ann.ID))
		gone, err := repo.FindByEmail(ctx, "ann@example.com")
		require.NoError(t, err)
		assert.Nil(t, gone)
		return nil
	}))
}
