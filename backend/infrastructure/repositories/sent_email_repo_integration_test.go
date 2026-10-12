//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSentEmailRepo_Integration_roundTripListAndDelete(t *testing.T) {
	// The app role, not the superuser: only it is subject to RLS.
	_, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewSentEmailRepo(db)
	in := func(fn func(ctx context.Context) error) { require.NoError(t, tenant.WithTenant(ctx, db, fn)) }
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	ruleID := "11111111-1111-1111-1111-111111111111"
	emails := []mcDomain.SentEmail{
		{TenantID: tenantID, Source: mcDomain.SentEmailSourceAutomation, Recipient: "ana@example.com", Subject: "Order #1001 confirmed",
			HTMLBody: "<p>Hi Ana</p>", Status: mcDomain.SentEmailSent, RuleID: ruleID, TriggerSubject: "storefront.order.placed", CreatedAt: base},
		{TenantID: tenantID, Source: mcDomain.SentEmailSourceNotification, Recipient: "team@example.com", Subject: "100% stock alert",
			HTMLBody: "<p>x</p>", Status: mcDomain.SentEmailFailed, Reason: "550 mailbox unavailable", CreatedAt: base.Add(time.Hour)},
		{TenantID: tenantID, Source: mcDomain.SentEmailSourceAutomation, Subject: "Stock report", Status: mcDomain.SentEmailSkipped,
			Reason: mcDomain.SkipNoRecipient, CreatedAt: base.Add(2 * time.Hour)},
	}
	in(func(ctx context.Context) error {
		for i := range emails {
			if err := repo.Create(ctx, &emails[i]); err != nil {
				return err
			}
		}
		return nil
	})
	require.NotEmpty(t, emails[0].ID)

	list := func(f mcDomain.SentEmailFilter) ([]mcDomain.SentEmail, int) {
		var out []mcDomain.SentEmail
		var total int
		in(func(ctx context.Context) error {
			var err error
			out, total, err = repo.List(ctx, f)
			return err
		})
		return out, total
	}

	all, total := list(mcDomain.SentEmailFilter{})
	require.Equal(t, 3, total)
	assert.Equal(t, []string{emails[2].ID, emails[1].ID, emails[0].ID}, []string{all[0].ID, all[1].ID, all[2].ID}, "newest first")
	assert.Empty(t, all[2].HTMLBody, "the list leaves bodies out")
	assert.Equal(t, ruleID, all[2].RuleID)

	page, total := list(mcDomain.SentEmailFilter{Page: 2, PerPage: 2})
	assert.Equal(t, 3, total)
	require.Len(t, page, 1)

	failed, _ := list(mcDomain.SentEmailFilter{Status: mcDomain.SentEmailFailed})
	require.Len(t, failed, 1)
	assert.Equal(t, "550 mailbox unavailable", failed[0].Reason)

	byText, _ := list(mcDomain.SentEmailFilter{Query: "ANA@"})
	require.Len(t, byText, 1)
	literal, _ := list(mcDomain.SentEmailFilter{Query: "100%"})
	require.Len(t, literal, 1, "% is matched literally")

	from := base.Add(30 * time.Minute)
	to := base.Add(90 * time.Minute)
	window, _ := list(mcDomain.SentEmailFilter{From: &from, To: &to})
	require.Len(t, window, 1)
	assert.Equal(t, emails[1].ID, window[0].ID)

	var found *mcDomain.SentEmail
	in(func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, emails[0].ID)
		return err
	})
	assert.Equal(t, "<p>Hi Ana</p>", found.HTMLBody)

	_, otherCtx := newTestTenant()
	require.NoError(t, tenant.WithTenant(otherCtx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, emails[0].ID)
		assert.ErrorIs(t, err, mcDomain.ErrSentEmailNotFound, "another tenant can't see it")
		return nil
	}))

	var deleted []string
	in(func(ctx context.Context) error {
		var err error
		deleted, err = repo.Delete(ctx, []string{emails[0].ID, emails[1].ID})
		return err
	})
	assert.ElementsMatch(t, []string{emails[0].ID, emails[1].ID}, deleted)
	_, total = list(mcDomain.SentEmailFilter{})
	assert.Equal(t, 1, total)
}
