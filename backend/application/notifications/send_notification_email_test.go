package notifications_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/messaging"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
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
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, client, "https://commerceos.example.nl", &stubTenantLanguage{})

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
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, client, "https://commerceos.example.nl", &stubTenantLanguage{})
	in := emailInput()
	in.Link = ""

	require.NoError(t, uc.Execute(tctx(), in))

	assert.NotContains(t, client.sent[0].HTMLBody, "href=")
}

func TestSendNotificationEmail_propagatesErrors(t *testing.T) {
	t.Run("connections", func(t *testing.T) {
		uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{err: errBoom}, "secret", unusedFactory{}, &recordingClient{}, "", &stubTenantLanguage{})
		assert.True(t, errors.Is(uc.Execute(tctx(), emailInput()), errBoom))
	})
	t.Run("send", func(t *testing.T) {
		uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, &recordingClient{err: errBoom}, "", &stubTenantLanguage{})
		assert.True(t, errors.Is(uc.Execute(tctx(), emailInput()), errBoom))
	})
}

type stubTenantLanguage struct {
	lang locale.Locale
	err  error
}

func (s *stubTenantLanguage) Execute(context.Context) (*settingsDomain.TenantSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &settingsDomain.TenantSettings{Language: s.lang}, nil
}

func TestSendNotificationEmail_translatedTypeUsesTheCompanyLanguage(t *testing.T) {
	client := &recordingClient{}
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, client, "https://commerceos.example.nl",
		&stubTenantLanguage{lang: locale.Dutch})

	err := uc.Execute(tctx(), appnotifications.SendNotificationEmailInput{
		Recipients: []string{"ops@example.nl"}, Type: notificationsDomain.TypeImportFailed, Params: map[string]any{"error": "Woo <500>"},
		Subject: "Import failed", Body: "Woo <500>", Link: "/sync-log",
	})

	require.NoError(t, err)
	m := client.sent[0]
	assert.Equal(t, "CommerceOS: Import mislukt", m.Subject)
	assert.Contains(t, m.HTMLBody, "Woo &lt;500&gt;")
	assert.Contains(t, m.HTMLBody, "Openen in CommerceOS")
	assert.Contains(t, m.HTMLBody, "meldingsinstellingen")
	assert.Equal(t, "Woo <500>", m.Body, "plain-text part translated too")
}

func TestSendNotificationEmail_untranslatedTypeKeepsItsTextWithTranslatedChrome(t *testing.T) {
	client := &recordingClient{}
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, client, "",
		&stubTenantLanguage{lang: locale.Portuguese})

	require.NoError(t, uc.Execute(tctx(), appnotifications.SendNotificationEmailInput{
		Recipients: []string{"ops@example.nl"}, Type: "something.new", Subject: "Hello", Body: "World",
	}))

	assert.Equal(t, "CommerceOS: Hello", client.sent[0].Subject)
	assert.Contains(t, client.sent[0].HTMLBody, "configurações de notificação")
}

func TestSendNotificationEmail_languageErrorPropagates(t *testing.T) {
	uc := appnotifications.NewSendNotificationEmailUseCase(newMockGormDB(t), &stubConnLister{}, "secret", unusedFactory{}, &recordingClient{}, "",
		&stubTenantLanguage{err: errBoom})

	assert.ErrorIs(t, uc.Execute(tctx(), emailInput()), errBoom)
}
