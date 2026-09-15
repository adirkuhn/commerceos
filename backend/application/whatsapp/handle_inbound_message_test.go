package whatsapp_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
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

// inboundEnvelope builds a minimal payload in the real (whatsmeow-based)
// shape Evolution API actually sends -- see
// TestHandleInboundMessageUseCase_parsesRealCapturedEvolutionAPIPayload
// for the full, verbatim-captured version this was confirmed against.
// remoteJID doubling as the isGroup signal (via its "@g.us" suffix) is a
// test-helper convenience only -- the real payload carries IsGroup as its
// own explicit boolean field, unrelated to the JID string.
func inboundEnvelope(remoteJID, text string, fromMe bool) webhooks.Envelope {
	isGroup := strings.HasSuffix(remoteJID, "@g.us")
	return webhooks.Envelope{
		EventID: "evt-1", TenantID: "tenant-1", Provider: "whatsapp",
		RawPayload: []byte(`{"data":{"Info":{"Sender":"` + remoteJID + `","IsFromMe":` + boolStr(fromMe) + `,"IsGroup":` + boolStr(isGroup) + `},"Message":{"conversation":"` + text + `"}}}`),
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
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123", StockQuantity: 7, CurrencyCode: "EUR"}},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Equal(t, "31612345678", client.sentTo)
	assert.Contains(t, client.sentBody, "Widget")
	assert.Contains(t, client.sentBody, "7")
}

func TestHandleInboundMessageUseCase_knownCustomer_greetsInReply(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
		&fakeCustomerRepo{byPhone: map[string]customersDomain.Customer{
			"31612345678": {ID: "cust-1", FirstName: "Jane"},
		}},
		nil,
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Contains(t, client.sentBody, "Hi Jane!")
}

func TestHandleInboundMessageUseCase_unknownNumber_repliesWithNoPersonalization(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31699999999@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.NotContains(t, client.sentBody, "Hi ")
}

func TestHandleInboundMessageUseCase_productNotFound_repliesGracefully(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{err: appwhatsapp.ErrProductNotFound},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ZZZ", false))

	require.NoError(t, err, "an unknown SKU code is a normal reply, not a dispatch error")
	assert.Contains(t, client.sentBody, "couldn't find")
}

func TestHandleInboundMessageUseCase_unrecognizedCommand_sendsHelpText(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil)

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
		&fakeMessageRepo{}, nil, nil)

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
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("120363025246285034@g.us", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Empty(t, client.sentTo, "a group's numeric id is not a phone number -- must never be dispatched as a direct-chat command or sent a reply")
}

// realCapturedEvolutionAPIMessagePayload is the exact (trimmed of
// irrelevant noise fields) webhook body Evolution API sent for a real
// inbound WhatsApp text message on 2026-09-15 (event ID
// 0000000000000000000000000000000A) -- captured directly from production
// logs after every inbound message turned out to be silently dropped.
// Sender is the gateway's own already-reconciled "<phone>@s.whatsapp.net"
// (SenderAlt carries the alternate "<numeric>@lid" identity for this
// particular LID-swap case, and must never be parsed as a phone number).
const realCapturedEvolutionAPIMessagePayload = `{
	"data": {
		"Info": {
			"Chat": "31600000001@s.whatsapp.net",
			"ID": "0000000000000000000000000000000A",
			"IsFromMe": false,
			"IsGroup": false,
			"PushName": "Adir Kuhn",
			"Sender": "31600000001@s.whatsapp.net",
			"SenderAlt": "100000000000001@lid",
			"Type": "text"
		},
		"Message": {
			"conversation": "Tem polvilho?",
			"messageContextInfo": {
				"deviceListMetadataVersion": 2,
				"messageSecret": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
			}
		}
	},
	"event": "Message",
	"instanceId": "00000000-0000-4000-8000-000000000001",
	"instanceName": "Cantinho1"
}`

func TestHandleInboundMessageUseCase_parsesRealCapturedEvolutionAPIPayload(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), webhooks.Envelope{
		EventID: "evt-1", TenantID: "tenant-1", Provider: "whatsapp",
		RawPayload: []byte(realCapturedEvolutionAPIMessagePayload),
	})

	require.NoError(t, err)
	assert.Equal(t, "31600000001", client.sentTo, "must dispatch to Sender's phone number, never SenderAlt's @lid identity")
	assert.Contains(t, client.sentBody, "Commands:", "unrecognized free text (no AI parsing configured in this test) falls back to help text -- proves the message was NOT silently dropped")
}

func TestHandleInboundMessageUseCase_logsInboundAndOutboundMessages(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	messageRepo := &fakeMessageRepo{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
		&fakeCustomerRepo{byPhone: map[string]customersDomain.Customer{
			"31612345678": {ID: "cust-1", FirstName: "Jane"},
		}},
		nil,
		messageRepo, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	require.Len(t, messageRepo.created, 2, "one inbound row and one outbound reply row")

	inbound := messageRepo.created[0]
	assert.Equal(t, whatsappDomain.MessageDirectionInbound, inbound.Direction)
	assert.Equal(t, "31612345678", inbound.PhoneE164)
	assert.Equal(t, "STOCK ABC123", inbound.Body)
	require.NotNil(t, inbound.CustomerID, "phone matched an existing customer -- must be linked")
	assert.Equal(t, "cust-1", *inbound.CustomerID)

	outbound := messageRepo.created[1]
	assert.Equal(t, whatsappDomain.MessageDirectionOutbound, outbound.Direction)
	assert.Equal(t, "31612345678", outbound.PhoneE164)
	assert.Equal(t, client.sentBody, outbound.Body)
	require.NotNil(t, outbound.CustomerID)
	assert.Equal(t, "cust-1", *outbound.CustomerID)
}

// Regression: NATS redelivers the same envelope whenever Execute returns
// an error (see infrastructure/consumers.WebhookConsumer -- the outer
// processed_events dedup only marks an event processed on full success),
// so a send failure legitimately retries the WHOLE handler from scratch.
// Before WebhookEventID-based dedup, that re-ran the unconditional inbound
// messageRepo.Create too, logging the same customer message once per
// retry -- a real customer conversation showed the same "Tem flocao"
// three times over. Only the inbound row must dedupe; the reply attempt
// itself must still retry every time (that's the whole point of NATS
// redelivery) until it succeeds.
func TestHandleInboundMessageUseCase_redeliveredEnvelopeAfterFailedSend_logsInboundOnlyOnce(t *testing.T) {
	client := &fakeEvolutionAPIClient{err: errors.New("evolutionapi: send text: status 401: not authorized")}
	messageRepo := &fakeMessageRepo{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 3), // 1st attempt: inbound only (send fails, no outbound tx). 2nd: inbound (deduped, no-op) + outbound.
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123"}},
		&fakeCustomerRepo{},
		nil,
		messageRepo, nil, nil,
	)
	envelope := inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false)

	err := uc.Execute(context.Background(), envelope)
	require.Error(t, err, "first attempt fails to send -- NATS will redeliver")
	require.Len(t, messageRepo.created, 1, "inbound logged once")

	client.err = nil // redelivery: the underlying issue (e.g. a since-fixed API key) is resolved
	err = uc.Execute(context.Background(), envelope)
	require.NoError(t, err, "redelivery retries the send and this time succeeds")

	require.Len(t, messageRepo.created, 2, "still just one inbound row -- deduped by WebhookEventID -- plus the one outbound reply")
	assert.Equal(t, whatsappDomain.MessageDirectionInbound, messageRepo.created[0].Direction)
	assert.Equal(t, whatsappDomain.MessageDirectionOutbound, messageRepo.created[1].Direction)
}

func TestHandleInboundMessageUseCase_logsMessageFromUnmatchedPhoneUnlinked(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	messageRepo := &fakeMessageRepo{}
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{},
		&fakeCustomerRepo{},
		nil,
		messageRepo, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31699999999@s.whatsapp.net", "hello there", false))

	require.NoError(t, err)
	require.Len(t, messageRepo.created, 2)
	assert.Nil(t, messageRepo.created[0].CustomerID, "no customer matched this phone -- still logged, just unlinked")
	assert.Nil(t, messageRepo.created[1].CustomerID)
}

func TestHandleInboundMessageUseCase_testModeRedirectsReplyToTestNumber(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	testPhone := "31699999999"
	conn := whatsappConnection(t)
	conn.WhatsAppTestModeEnabled = true
	conn.WhatsAppTestPhoneE164 = &testPhone
	uc := appwhatsapp.NewHandleInboundMessageUseCase(
		newMockGormDB(t, 2),
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{conn}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
		&fakeProductInfoLookup{info: &appwhatsapp.ProductInfo{ProductName: "Widget", SKUCode: "ABC123", StockQuantity: 7, CurrencyCode: "EUR"}},
		&fakeCustomerRepo{},
		nil,
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.NoError(t, err)
	assert.Equal(t, testPhone, client.sentTo, "test mode must redirect to the connection's test number, never the real customer")
	assert.Contains(t, client.sentBody, "31612345678", "the real intended recipient must still be identifiable in the redirected body")
	assert.Contains(t, client.sentBody, "Widget")
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
		&fakeMessageRepo{}, nil, nil)

	err := uc.Execute(context.Background(), inboundEnvelope("31612345678@s.whatsapp.net", "STOCK ABC123", false))

	require.Error(t, err, "an infra failure behind the lookup must surface as a dispatch error (nack/retry), not a false 'not found' reply")
	assert.Empty(t, client.sentTo)
}
