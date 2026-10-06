package whatsapp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

func newInboundUC(t *testing.T, client *fakeEvolutionAPIClient, outbox *eventrecordtest.Outbox) *appwhatsapp.HandleInboundMessageUseCase {
	return appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123", StockQuantity: 7, CurrencyCode: "EUR"}},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil, outbox)
}

func TestHandleInboundMessage_RecordsReceivedAndSent(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	require.NoError(t, newInboundUC(t, &fakeEvolutionAPIClient{}, outbox).Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false)))

	require.Equal(t, []string{"whatsapp.message.received", "whatsapp.message.sent"}, outbox.Subjects())
	assert.Equal(t, "msg-1", outbox.Events[0].Payload["entity_id"])
	assert.Equal(t, "msg-2", outbox.Events[1].Payload["entity_id"])
	assert.Contains(t, outbox.Events[0].Data(), "webhook_event_id")
	for _, e := range outbox.Events {
		assert.NotContains(t, e.Data(), "body", "message text stays out of events")
	}
}

// A redelivered webhook isn't logged twice (the repo dedupes it), so no
// second received event either.
func TestHandleInboundMessage_DuplicateDeliveryRecordsNoSecondReceived(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &fakeMessageRepo{}
	envelope := inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false)
	mk := func() *appwhatsapp.HandleInboundMessageUseCase {
		return appwhatsapp.NewHandleInboundMessageUseCase(newMockGormDB(t, 2),
			&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
			&fakeEvolutionAPIClientFactory{client: &fakeEvolutionAPIClient{}}, testEncryptionSecret,
			&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
			&fakeCustomerRepo{}, nil, repo, nil, nil, outbox)
	}

	require.NoError(t, mk().Execute(context.Background(), envelope))
	require.NoError(t, mk().Execute(context.Background(), envelope))

	received := 0
	for _, s := range outbox.Subjects() {
		if s == "whatsapp.message.received" {
			received++
		}
	}
	assert.Equal(t, 1, received)
}

func TestHandleInboundMessage_FailsWhenTheReceivedEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	client := &fakeEvolutionAPIClient{}

	err := newInboundUC(t, client, &eventrecordtest.Outbox{Err: boom}).Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	assert.ErrorIs(t, err, boom)
	assert.Empty(t, client.sentTo, "no reply when the inbound message didn't commit")
}

func TestOrderFlow_RecordsConversationUpdated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appwhatsapp.NewOrderFlowUseCase(newMockGormDB(t, 2), &fakeConversationRepo{}, orderFlowChannelRepo(),
		&fakeProductInfoLookup{info: ptr(widgetInfo())}, &fakeSalesOrderCreator{}, &fakeCustomerCreator{}, nil, outbox)

	_, _, err := uc.Execute(context.Background(), testPhone, nil, "ORDER ABC123 2")

	require.NoError(t, err)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "whatsapp.conversation.updated", e.Subject)
	assert.Equal(t, testPhone, e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"state": string(whatsappDomain.ConversationStateIdle), "cart_item_count": float64(1)}, e.Data())
}

func TestDeleteWhatsAppConversation_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	require.NoError(t, appwhatsapp.NewDeleteWhatsAppConversationUseCase(newMockGormDB(t, 1), &fakeMessageRepo{}, outbox).Execute(context.Background(), "31612345678"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "whatsapp.conversation.deleted", e.Subject)
	assert.Equal(t, "31612345678", e.Payload["entity_id"])
}
