package storefront_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func redeemableMagicLink() storefrontDomain.MagicLinkToken {
	return storefrontDomain.MagicLinkToken{
		ID: "magic-link-1", TenantID: "tenant-1", SalesChannelID: "channel-1", CustomerID: "customer-1",
		Token: "LINK123", ExpiresAt: time.Now().Add(10 * time.Minute),
	}
}

func TestVerifyMagicLink_exchangesARedeemableTokenForASession(t *testing.T) {
	tokenRepo := &fakeMagicLinkTokenRepo{byToken: map[string]storefrontDomain.MagicLinkToken{"LINK123": redeemableMagicLink()}}
	sessionRepo := &fakeCustomerSessionRepo{}
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), tokenRepo, sessionRepo, &eventrecordtest.Outbox{})

	session, err := uc.Execute(context.Background(), "channel-1", "LINK123")

	require.NoError(t, err)
	require.Equal(t, "customer-1", session.CustomerID)
	require.Equal(t, "channel-1", session.SalesChannelID)
	require.NotEmpty(t, session.Token)
	require.Equal(t, "magic-link-1", tokenRepo.markUsedID)
	require.Equal(t, sessionRepo.created, session)
}

func TestVerifyMagicLink_rejectsUnknownToken(t *testing.T) {
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), &fakeMagicLinkTokenRepo{}, &fakeCustomerSessionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "channel-1", "missing-token")

	require.ErrorIs(t, err, storefrontDomain.ErrInvalidOrExpiredMagicLink)
}

func TestVerifyMagicLink_rejectsExpiredToken(t *testing.T) {
	expired := redeemableMagicLink()
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	tokenRepo := &fakeMagicLinkTokenRepo{byToken: map[string]storefrontDomain.MagicLinkToken{"LINK123": expired}}
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), tokenRepo, &fakeCustomerSessionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "channel-1", "LINK123")

	require.ErrorIs(t, err, storefrontDomain.ErrInvalidOrExpiredMagicLink)
}

func TestVerifyMagicLink_rejectsAlreadyUsedToken(t *testing.T) {
	used := redeemableMagicLink()
	usedAt := time.Now().Add(-time.Minute)
	used.UsedAt = &usedAt
	tokenRepo := &fakeMagicLinkTokenRepo{byToken: map[string]storefrontDomain.MagicLinkToken{"LINK123": used}}
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), tokenRepo, &fakeCustomerSessionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "channel-1", "LINK123")

	require.ErrorIs(t, err, storefrontDomain.ErrInvalidOrExpiredMagicLink)
}

func TestVerifyMagicLink_rejectsWrongChannel(t *testing.T) {
	tokenRepo := &fakeMagicLinkTokenRepo{byToken: map[string]storefrontDomain.MagicLinkToken{"LINK123": redeemableMagicLink()}}
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), tokenRepo, &fakeCustomerSessionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "channel-2", "LINK123")

	require.ErrorIs(t, err, storefrontDomain.ErrInvalidOrExpiredMagicLink)
}

func TestVerifyMagicLink_raceOnMarkUsedFailsTheWholeExchange(t *testing.T) {
	tokenRepo := &fakeMagicLinkTokenRepo{
		byToken:     map[string]storefrontDomain.MagicLinkToken{"LINK123": redeemableMagicLink()},
		markUsedErr: storefrontDomain.ErrInvalidOrExpiredMagicLink,
	}
	sessionRepo := &fakeCustomerSessionRepo{}
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), tokenRepo, sessionRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "channel-1", "LINK123")

	require.ErrorIs(t, err, storefrontDomain.ErrInvalidOrExpiredMagicLink)
	require.Nil(t, sessionRepo.created)
}
