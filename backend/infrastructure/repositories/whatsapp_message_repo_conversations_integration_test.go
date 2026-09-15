//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestWhatsAppMessageRepo_Integration_ListConversations(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewWhatsAppMessageRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)

	var customerID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c := &customersDomain.Customer{TenantID: tenantID, Email: "jane@example.com", FirstName: "Jane", LastName: "Doe"}
		if err := customerRepo.Create(ctx, c); err != nil {
			return err
		}
		customerID = c.ID

		// Phone A (linked to Jane): two messages, second is newer.
		if err := repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31612345678", CustomerID: &customerID,
			Direction: whatsappDomain.MessageDirectionInbound, Body: "STOCK ABC123",
		}); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		if err := repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31612345678", CustomerID: &customerID,
			Direction: whatsappDomain.MessageDirectionOutbound, Body: "Widget - in stock: 7",
		}); err != nil {
			return err
		}

		// Phone B (never matched a customer): one message, oldest overall.
		time.Sleep(10 * time.Millisecond)
		return repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31699999999",
			Direction: whatsappDomain.MessageDirectionInbound, Body: "hello",
		})
	})
	require.NoError(t, err)

	var summaries []whatsappDomain.ConversationSummary
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		summaries, err = repo.ListConversations(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, summaries, 2, "one row per phone number, not per message")

	// Newest conversation first -- phone B's single "hello" message is the
	// most recent event overall.
	require.Equal(t, "31699999999", summaries[0].PhoneE164)
	require.Nil(t, summaries[0].CustomerID)
	require.Equal(t, "", summaries[0].CustomerName)
	require.Equal(t, "hello", summaries[0].LastBody)

	require.Equal(t, "31612345678", summaries[1].PhoneE164)
	require.NotNil(t, summaries[1].CustomerID)
	require.Equal(t, customerID, *summaries[1].CustomerID)
	require.Equal(t, "Jane Doe", summaries[1].CustomerName)
	require.Equal(t, "Widget - in stock: 7", summaries[1].LastBody, "the phone's MOST RECENT message, not its first")
	require.Equal(t, whatsappDomain.MessageDirectionOutbound, summaries[1].LastDirection)
}

func TestWhatsAppMessageRepo_Integration_ListConversations_isolatesTenants(t *testing.T) {
	_, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	tenantA, ctxA := newTestTenant()
	tenantB, ctxB := newTestTenant()
	repo := repositories.NewWhatsAppMessageRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantA, PhoneE164: "31612345678", Direction: whatsappDomain.MessageDirectionInbound, Body: "tenant a",
		})
	})
	require.NoError(t, err)
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		return repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantB, PhoneE164: "31699999999", Direction: whatsappDomain.MessageDirectionInbound, Body: "tenant b",
		})
	})
	require.NoError(t, err)

	var summariesA []whatsappDomain.ConversationSummary
	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		var err error
		summariesA, err = repo.ListConversations(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, summariesA, 1)
	require.Equal(t, "31612345678", summariesA[0].PhoneE164)
}
