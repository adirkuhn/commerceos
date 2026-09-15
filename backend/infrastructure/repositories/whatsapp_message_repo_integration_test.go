//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestWhatsAppMessageRepo_Integration_CreateAndListByPhoneAndByCustomer(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewWhatsAppMessageRepo(db)
	customerID := uuid.NewString()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		inbound := &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31612345678", CustomerID: &customerID,
			Direction: whatsappDomain.MessageDirectionInbound, Body: "STOCK ABC123",
		}
		if err := repo.Create(ctx, inbound); err != nil {
			return err
		}
		outbound := &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31612345678", CustomerID: &customerID,
			Direction: whatsappDomain.MessageDirectionOutbound, Body: "Widget - in stock: 7",
		}
		return repo.Create(ctx, outbound)
	})
	require.NoError(t, err)

	var byPhone []whatsappDomain.Message
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byPhone, err = repo.ListByPhone(ctx, "31612345678")
		return err
	})
	require.NoError(t, err)
	require.Len(t, byPhone, 2)
	require.Equal(t, whatsappDomain.MessageDirectionOutbound, byPhone[0].Direction, "newest first")

	var byCustomer []whatsappDomain.Message
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byCustomer, err = repo.ListByCustomer(ctx, customerID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, byCustomer, 2)
}

func TestWhatsAppMessageRepo_Integration_UnmatchedPhoneHasNilCustomerID(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewWhatsAppMessageRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31699999999",
			Direction: whatsappDomain.MessageDirectionInbound, Body: "hello",
		})
	})
	require.NoError(t, err)

	var byPhone []whatsappDomain.Message
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		byPhone, err = repo.ListByPhone(ctx, "31699999999")
		return err
	})
	require.NoError(t, err)
	require.Len(t, byPhone, 1)
	require.Nil(t, byPhone[0].CustomerID)
}

func TestWhatsAppMessageRepo_Integration_DeleteByPhone(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	repo := repositories.NewWhatsAppMessageRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31612345678", Direction: whatsappDomain.MessageDirectionInbound, Body: "hi",
		}); err != nil {
			return err
		}
		return repo.Create(ctx, &whatsappDomain.Message{
			TenantID: tenantID, PhoneE164: "31699999999", Direction: whatsappDomain.MessageDirectionInbound, Body: "oi",
		})
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.DeleteByPhone(ctx, "31612345678")
	})
	require.NoError(t, err)

	var deleted, untouched []whatsappDomain.Message
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if deleted, err = repo.ListByPhone(ctx, "31612345678"); err != nil {
			return err
		}
		untouched, err = repo.ListByPhone(ctx, "31699999999")
		return err
	})
	require.NoError(t, err)
	require.Empty(t, deleted)
	require.Len(t, untouched, 1)
}

func TestWhatsAppMessageRepo_Integration_DeleteByPhone_neverExistedIsNotAnError(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewWhatsAppMessageRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.DeleteByPhone(ctx, "31600000000")
	})
	require.NoError(t, err)
}
