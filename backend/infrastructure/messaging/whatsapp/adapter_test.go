package whatsapp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/whatsapp"
)

type fakeEvolutionClient struct {
	sentTo, sentBody string
	err              error
}

func (f *fakeEvolutionClient) SendText(_ context.Context, toPhone, body string) error {
	f.sentTo, f.sentBody = toPhone, body
	return f.err
}

func TestClientAdapter_Send_delegatesToEvolutionClient(t *testing.T) {
	evo := &fakeEvolutionClient{}
	adapter := whatsapp.NewClientAdapter(evo)

	err := adapter.Send(context.Background(), messaging.Message{Channel: messaging.ChannelWhatsApp, To: "31600000000", Body: "hi"})

	require.NoError(t, err)
	require.Equal(t, "31600000000", evo.sentTo)
	require.Equal(t, "hi", evo.sentBody)
}

func TestClientAdapter_Send_propagatesEvolutionClientError(t *testing.T) {
	evo := &fakeEvolutionClient{err: errors.New("evolution api down")}
	adapter := whatsapp.NewClientAdapter(evo)

	err := adapter.Send(context.Background(), messaging.Message{Channel: messaging.ChannelWhatsApp, To: "31600000000", Body: "hi"})

	require.Error(t, err)
}

func TestClientAdapter_Send_rejectsNonWhatsAppChannel(t *testing.T) {
	adapter := whatsapp.NewClientAdapter(&fakeEvolutionClient{})

	err := adapter.Send(context.Background(), messaging.Message{Channel: messaging.ChannelEmail, To: "shopper@example.com"})

	require.ErrorContains(t, err, "unsupported channel")
}
