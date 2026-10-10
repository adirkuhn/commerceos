package messagecenter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	"github.com/adirkuhn/commerceos/domain/locale"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

func dutchSettings() *stubTenantSettings {
	return &stubTenantSettings{settings: &settingsDomain.TenantSettings{Language: locale.Dutch}}
}

func TestListMessageTemplateDefaults_inTheTenantsLanguage(t *testing.T) {
	uc := appmc.NewListMessageTemplateDefaultsUseCase(dutchSettings())

	defaults, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, mcDomain.DefaultTemplates(locale.Dutch), defaults)
}

func TestListMessageTemplateDefaults_settingsError(t *testing.T) {
	boom := errors.New("boom")
	_, err := appmc.NewListMessageTemplateDefaultsUseCase(&stubTenantSettings{err: boom}).Execute(context.Background())
	assert.ErrorIs(t, err, boom)
}

func TestCreateMessageTemplate_remembersTheDefaultItCameFrom(t *testing.T) {
	repo := &stubMessageTemplateRepo{}
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "Order", Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{messaging.ChannelEmail},
		Body: "<p>Hi</p>", DefaultKey: mcDomain.DefaultOrderConfirmation,
	})

	require.NoError(t, err)
	assert.Equal(t, mcDomain.DefaultOrderConfirmation, repo.created[0].DefaultKey)
}

func TestCreateMessageTemplate_rejectsAnUnknownDefault(t *testing.T) {
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "Order", Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{messaging.ChannelEmail},
		Body: "<p>Hi</p>", DefaultKey: "nope",
	})

	assert.ErrorIs(t, err, mcDomain.ErrUnknownDefaultTemplate)
}

func customizedTemplate() mcDomain.MessageTemplate {
	return mcDomain.MessageTemplate{
		ID: "tpl-1", Name: "Mine", Category: mcDomain.CategoryPurchase,
		Channels: []messaging.Channel{messaging.ChannelEmail}, Subject: "custom", Body: "<p>custom</p>", TextBody: "custom",
		Design: []byte(`{"pages":[]}`), DefaultKey: mcDomain.DefaultOrderConfirmation, Enabled: false,
	}
}

func TestResetMessageTemplate_restoresTheDefaultContent(t *testing.T) {
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"tpl-1": customizedTemplate()}}
	outbox := &eventrecordtest.Outbox{}
	uc := appmc.NewResetMessageTemplateUseCase(newMockGormDB(t, 1), repo, outbox, dutchSettings())

	tmpl, err := uc.Execute(context.Background(), "tpl-1")

	require.NoError(t, err)
	def, _ := mcDomain.DefaultTemplateFor(mcDomain.DefaultOrderConfirmation, locale.Dutch)
	require.Len(t, repo.updated, 1)
	got := repo.updated[0]
	assert.Equal(t, def.Template.Subject, got.Subject)
	assert.Equal(t, def.Template.Body, got.Body)
	assert.Equal(t, def.Template.TextBody, got.TextBody)
	assert.Nil(t, got.Design, "the visual design restarts from the default html")
	assert.Equal(t, "Mine", got.Name, "name, channels and enabled stay the tenant's")
	assert.Equal(t, []messaging.Channel{messaging.ChannelEmail}, got.Channels)
	assert.False(t, got.Enabled)
	assert.Equal(t, got, *tmpl)
	assert.Len(t, outbox.Events, 1)
}

func TestResetMessageTemplate_withoutADefault(t *testing.T) {
	own := customizedTemplate()
	own.DefaultKey = ""
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"tpl-1": own}}
	uc := appmc.NewResetMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{}, dutchSettings())

	_, err := uc.Execute(context.Background(), "tpl-1")

	assert.ErrorIs(t, err, mcDomain.ErrTemplateHasNoDefault)
	assert.Empty(t, repo.updated)
}

func TestResetMessageTemplate_notFound(t *testing.T) {
	uc := appmc.NewResetMessageTemplateUseCase(newMockGormDB(t, 1), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{}, dutchSettings())

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNotFound)
}

func TestResetMessageTemplate_settingsError(t *testing.T) {
	boom := errors.New("boom")
	uc := appmc.NewResetMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{}, &stubTenantSettings{err: boom})

	_, err := uc.Execute(context.Background(), "tpl-1")

	assert.ErrorIs(t, err, boom)
}
