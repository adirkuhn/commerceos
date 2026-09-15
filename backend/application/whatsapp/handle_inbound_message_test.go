package whatsapp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

var assertAnError = errors.New("infra failure")

const testEncryptionSecret = "test-encryption-secret"

func encryptedSecret(t *testing.T, plaintext string) string {
	t.Helper()
	enc, err := crypto.Encrypt(testEncryptionSecret, plaintext)
	require.NoError(t, err)
	return enc
}

func whatsappConnection(t *testing.T) integrationsDomain.Connection {
	return integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp,
		StoreURL: "https://evolution.example.com", ConsumerKey: "tenant-1-instance",
		ConsumerSecretEncrypted: encryptedSecret(t, "api-key-plain"),
	}
}

func inboundEnvelope(remoteJID, text string, fromMe bool) webhooks.Envelope {
	return webhooks.Envelope{
		EventID: "evt-1", TenantID: "tenant-1", Provider: "whatsapp",
		RawPayload: []byte(`{"data":{"key":{"remoteJid":"` + remoteJID + `","fromMe":` + boolStr(fromMe) + `},"message":{"conversation":"` + text + `"}}}`),
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestHandleInboundMessageUseCase_stockCommand_sendsFormattedReply(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 1),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123", StockQuantity: 7, CurrencyCode: "EUR"}},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Equal(t, "31612345678", client.sentTo)
	assert.Contains(t, client.sentBody, "Widget")
	assert.Contains(t, client.sentBody, "7")
}

func TestHandleInboundMessageUseCase_knownCustomer_greetsInReply(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 1),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
		&fakeCustomerRepo{byPhone: map[string]customersDomain.Customer{
			"31612345678": {ID: "cust-1", FirstName: "Jane"},
		}},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Contains(t, client.sentBody, "Hi Jane!")
}

func TestHandleInboundMessageUseCase_unknownNumber_repliesWithNoPersonalization(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 1),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31699999999@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.NotContains(t, client.sentBody, "Hi ")
}

func TestHandleInboundMessageUseCase_productNotFound_repliesGracefully(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 1),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{err: appwhatsapp.ErrProductNotFound},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ZZZ", false))

	require.NoError(t, err, "an unknown SKU code is a normal reply, not a dispatch error")
	assert.Contains(t, client.sentBody, "couldn't find")
}

func TestHandleInboundMessageUseCase_unrecognizedCommand_sendsHelpText(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 1),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "hello there", false))

	require.NoError(t, err)
	assert.Contains(t, client.sentBody, "Commands:")
}

func TestHandleInboundMessageUseCase_fromMeMessage_neverDispatched(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 0),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", true))

	require.NoError(t, err)
	assert.Empty(t, client.sentTo, "our own outbound message echoed back must never be treated as an inbound command")
}

func TestHandleInboundMessageUseCase_groupChatMessage_neverDispatched(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 0),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("120363025246285034@g.us", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Empty(t, client.sentTo, "a group's numeric id is not a phone number -- must never be dispatched as a direct-chat command or sent a reply")
}

func TestHandleInboundMessageUseCase_lookupInfraError_propagates(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 1),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{err: assertAnError},
		&fakeCustomerRepo{},
		nil,
	)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.Error(t, err, "an infra failure behind the lookup must surface as a dispatch error (nack/retry), not a false 'not found' reply")
	assert.Empty(t, client.sentTo)
}
