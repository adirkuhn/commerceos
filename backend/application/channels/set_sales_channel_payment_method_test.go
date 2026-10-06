package channels_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestSetSalesChannelPaymentMethod_validStripeConnection_enablesIt(t *testing.T) {
	channel := &channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp}
	repo := &stubSalesChannelRepo{findResult: channel}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-stripe-1", Provider: integrationsDomain.ProviderStripe, Status: integrationsDomain.StatusConnected}}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, connRepo, &eventrecordtest.Outbox{})

	connID := "conn-stripe-1"
	result, err := uc.Execute(context.Background(), "channel-1", &connID)

	require.NoError(t, err)
	require.NotNil(t, result.PaymentConnectionID)
	assert.Equal(t, "conn-stripe-1", *result.PaymentConnectionID)
	require.Len(t, repo.updated, 1)
	require.NotNil(t, repo.updated[0].PaymentConnectionID)
	assert.Equal(t, "conn-stripe-1", *repo.updated[0].PaymentConnectionID)
}

func TestSetSalesChannelPaymentMethod_nilDisablesIt(t *testing.T) {
	existingConnID := "conn-stripe-1"
	channel := &channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp, PaymentConnectionID: &existingConnID}
	repo := &stubSalesChannelRepo{findResult: channel}
	connRepo := &stubConnectionRepo{}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, connRepo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "channel-1", nil)

	require.NoError(t, err)
	assert.Nil(t, result.PaymentConnectionID)
	require.Len(t, repo.updated, 1)
	assert.Nil(t, repo.updated[0].PaymentConnectionID)
}

func TestSetSalesChannelPaymentMethod_nonPaymentProvider_rejects(t *testing.T) {
	channel := &channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp}
	repo := &stubSalesChannelRepo{findResult: channel}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-woo-1", Provider: integrationsDomain.ProviderWooCommerce, Status: integrationsDomain.StatusConnected}}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, connRepo, &eventrecordtest.Outbox{})

	connID := "conn-woo-1"
	_, err := uc.Execute(context.Background(), "channel-1", &connID)

	require.ErrorIs(t, err, channelsDomain.ErrInvalidPaymentConnection)
	assert.Empty(t, repo.updated)
}

func TestSetSalesChannelPaymentMethod_disconnectedPaymentProvider_rejects(t *testing.T) {
	channel := &channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp}
	repo := &stubSalesChannelRepo{findResult: channel}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-stripe-1", Provider: integrationsDomain.ProviderStripe, Status: integrationsDomain.StatusError}}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, connRepo, &eventrecordtest.Outbox{})

	connID := "conn-stripe-1"
	_, err := uc.Execute(context.Background(), "channel-1", &connID)

	require.ErrorIs(t, err, channelsDomain.ErrInvalidPaymentConnection)
}

func TestSetSalesChannelPaymentMethod_unknownConnection_rejects(t *testing.T) {
	channel := &channelsDomain.SalesChannel{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp}
	repo := &stubSalesChannelRepo{findResult: channel}
	connRepo := &stubConnectionRepo{findErr: integrationsDomain.ErrConnectionNotFound}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, connRepo, &eventrecordtest.Outbox{})

	connID := "does-not-exist"
	_, err := uc.Execute(context.Background(), "channel-1", &connID)

	require.ErrorIs(t, err, channelsDomain.ErrInvalidPaymentConnection)
}

func TestSetSalesChannelPaymentMethod_channelNotFound_propagates(t *testing.T) {
	repo := &stubSalesChannelRepo{findErr: channelsDomain.ErrSalesChannelNotFound}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "channel-1", nil)

	require.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}
