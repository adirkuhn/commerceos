package notifications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

type stubConnLister struct {
	conns []integrationsDomain.Connection
	err   error
}

func (s *stubConnLister) ListByTenant(context.Context) ([]integrationsDomain.Connection, error) {
	return s.conns, s.err
}

type recordingClient struct {
	sent []messaging.Message
	err  error
}

func (c *recordingClient) Send(_ context.Context, m messaging.Message) error {
	if c.err != nil {
		return c.err
	}
	c.sent = append(c.sent, m)
	return nil
}

type unusedFactory struct{}

func (unusedFactory) NewClient(string, string, string, string, string) messaging.Client { return nil }

func emailInput() appnotifications.SendNotificationEmailInput {
	return appnotifications.SendNotificationEmailInput{
		Recipients: []string{"ops@example.nl", "boss@example.nl"},
		Subject:    "Import failed", Body: "Woo returned <500>", Link: "/sync-log",
	}
}

func TestSendNotificationEmail_sendsOneEmailPerRecipient(t *testing.T) {
	client := &recordingClient{}
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, client, "https://commerceos.example.nl")

	require.NoError(t, uc.Execute(tctx(), emailInput()))

	require.Len(t, client.sent, 2)
	m := client.sent[0]
	assert.Equal(t, messaging.ChannelEmail, m.Channel)
	assert.Equal(t, "ops@example.nl", m.To)
	assert.Equal(t, "CommerceOS: Import failed", m.Subject)
	assert.Contains(t, m.HTMLBody, "Woo returned &lt;500&gt;", "message text is HTML-escaped")
	assert.Contains(t, m.HTMLBody, `href="https://commerceos.example.nl/sync-log"`)
	assert.Equal(t, "boss@example.nl", client.sent[1].To)
}

func TestSendNotificationEmail_noLinkNoButton(t *testing.T) {
	client := &recordingClient{}
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, client, "https://commerceos.example.nl")
	in := emailInput()
	in.Link = ""

	require.NoError(t, uc.Execute(tctx(), in))

	assert.NotContains(t, client.sent[0].HTMLBody, "href=")
}

func TestSendNotificationEmail_propagatesErrors(t *testing.T) {
	t.Run("connections", func(t *testing.T) {
		uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{err: errBoom}, "secret", unusedFactory{}, &recordingClient{}, "")
		assert.True(t, errors.Is(uc.Execute(tctx(), emailInput()), errBoom))
	})
	t.Run("send", func(t *testing.T) {
		uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, &recordingClient{err: errBoom}, "")
		assert.True(t, errors.Is(uc.Execute(tctx(), emailInput()), errBoom))
	})
}
