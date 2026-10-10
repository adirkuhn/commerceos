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
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubTenantSettings struct {
	settings *settingsDomain.TenantSettings
	err      error
}

func (s *stubTenantSettings) Execute(context.Context) (*settingsDomain.TenantSettings, error) {
	return s.settings, s.err
}

func TestEnsureStockExpiryDefaults_createsTemplateAndRuleInTheTenantLanguage(t *testing.T) {
	rules, templates, outbox := &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{}
	uc := appmc.NewEnsureStockExpiryDefaultsUseCase(newMockGormDB(t, 1), rules, templates, outbox,
		&stubTenantSettings{settings: &settingsDomain.TenantSettings{Language: locale.Dutch}})

	require.NoError(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1")))

	want := mcDomain.DefaultStockExpiry(locale.Dutch)
	require.Len(t, templates.created, 1)
	assert.Equal(t, want.Template.Name, templates.created[0].Name)
	assert.Equal(t, want.Template.Body, templates.created[0].Body)
	assert.Equal(t, "tenant-1", templates.created[0].TenantID)
	require.Len(t, rules.created, 1)
	assert.Equal(t, mcDomain.TriggerStockCloseToExpiry, rules.created[0].TriggerSubject)
	assert.Equal(t, templates.created[0].ID, rules.created[0].TemplateID)
	assert.Equal(t, []string{}, rules.created[0].ExtraRecipients)
	assert.Equal(t, []string{"messagecenter.message_template.created", "messagecenter.automation_rule.created"}, outbox.Subjects())
}

func TestEnsureStockExpiryDefaults_keepsAnExistingRule(t *testing.T) {
	rules := &stubAutomationRuleRepo{all: []mcDomain.AutomationRule{{ID: "r-9", TriggerSubject: mcDomain.TriggerStockCloseToExpiry, Enabled: false}}}
	templates, outbox := &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{}
	uc := appmc.NewEnsureStockExpiryDefaultsUseCase(newMockGormDB(t, 1), rules, templates, outbox, &stubTenantSettings{settings: &settingsDomain.TenantSettings{}})

	require.NoError(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1")))

	assert.Empty(t, templates.created, "a tenant who already has (even a disabled) rule keeps it")
	assert.Empty(t, rules.created)
	assert.Empty(t, outbox.Events)
}

func TestEnsureStockExpiryDefaults_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")

	uc := appmc.NewEnsureStockExpiryDefaultsUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{}, &stubTenantSettings{err: boom})
	assert.ErrorIs(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1")), boom)

	uc = appmc.NewEnsureStockExpiryDefaultsUseCase(newMockGormDB(t, 1), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{Err: boom},
		&stubTenantSettings{settings: &settingsDomain.TenantSettings{}})
	assert.ErrorIs(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1")), boom)
}
