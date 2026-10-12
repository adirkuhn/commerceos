package messagecenter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	"github.com/adirkuhn/commerceos/domain/locale"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubSetupRepo struct {
	seeded   bool
	marked   int
	unseeded []string
	err      error
}

func (s *stubSetupRepo) Claim(context.Context, time.Time) (bool, error) {
	if s.err != nil || s.seeded {
		return false, s.err
	}
	s.marked++
	s.seeded = true
	return true, nil
}
func (s *stubSetupRepo) ListUnseededTenants(context.Context) ([]string, error) {
	return s.unseeded, s.err
}

func TestEnsureMessageCenterDefaults_createsEveryDefaultTemplateAndRule(t *testing.T) {
	templates, rules, setup, outbox := &stubMessageTemplateRepo{}, &stubAutomationRuleRepo{}, &stubSetupRepo{}, &eventrecordtest.Outbox{}
	uc := appmc.NewEnsureMessageCenterDefaultsUseCase(newMockGormDB(t, 1), templates, rules, setup, outbox, dutchSettings())

	require.NoError(t, uc.Execute(tenant.AsTenant(context.Background(), "tenant-1")))

	defaults := mcDomain.DefaultTemplates(locale.Dutch)
	require.Len(t, templates.created, len(defaults))
	require.Len(t, rules.created, len(defaults))
	for i, d := range defaults {
		assert.Equal(t, d.Template.Name, templates.created[i].Name)
		assert.Equal(t, d.Key, templates.created[i].DefaultKey)
		assert.True(t, templates.created[i].Enabled)
		assert.Equal(t, d.TriggerSubject, rules.created[i].TriggerSubject)
		assert.Equal(t, "template-1", rules.created[i].TemplateID, "rule points at its template")
		assert.True(t, rules.created[i].Enabled)
		assert.Equal(t, "tenant-1", rules.created[i].TenantID)
	}
	assert.Equal(t, 1, setup.marked)
	assert.Len(t, outbox.Events, 2*len(defaults), "every template and rule raises its created event")
}

func TestEnsureMessageCenterDefaults_runsOncePerTenant(t *testing.T) {
	templates, rules := &stubMessageTemplateRepo{}, &stubAutomationRuleRepo{}
	uc := appmc.NewEnsureMessageCenterDefaultsUseCase(newMockGormDB(t, 1), templates, rules, &stubSetupRepo{seeded: true}, &eventrecordtest.Outbox{}, dutchSettings())

	require.NoError(t, uc.Execute(tenant.AsTenant(context.Background(), "tenant-1")))

	assert.Empty(t, templates.created, "deleted defaults don't come back")
	assert.Empty(t, rules.created)
}

func TestEnsureMessageCenterDefaults_skipsTriggersTheTenantAlreadyHandles(t *testing.T) {
	templates := &stubMessageTemplateRepo{}
	rules := &stubAutomationRuleRepo{all: []mcDomain.AutomationRule{{TriggerSubject: mcDomain.TriggerStockCloseToExpiry}}}
	uc := appmc.NewEnsureMessageCenterDefaultsUseCase(newMockGormDB(t, 1), templates, rules, &stubSetupRepo{}, &eventrecordtest.Outbox{}, dutchSettings())

	require.NoError(t, uc.Execute(tenant.AsTenant(context.Background(), "tenant-1")))

	assert.Len(t, templates.created, len(mcDomain.DefaultTemplates(locale.Dutch))-1)
	for _, r := range rules.created {
		assert.NotEqual(t, mcDomain.TriggerStockCloseToExpiry, r.TriggerSubject)
	}
}

func TestEnsureMessageCenterDefaults_errors(t *testing.T) {
	boom := errors.New("boom")
	uc := appmc.NewEnsureMessageCenterDefaultsUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &stubAutomationRuleRepo{}, &stubSetupRepo{}, &eventrecordtest.Outbox{}, &stubTenantSettings{err: boom})
	assert.ErrorIs(t, uc.Execute(tenant.AsTenant(context.Background(), "tenant-1")), boom)

	uc = appmc.NewEnsureMessageCenterDefaultsUseCase(newMockGormDB(t, 1), &stubMessageTemplateRepo{}, &stubAutomationRuleRepo{}, &stubSetupRepo{err: boom}, &eventrecordtest.Outbox{}, dutchSettings())
	assert.ErrorIs(t, uc.Execute(tenant.AsTenant(context.Background(), "tenant-1")), boom)
}

type recordingEnsurer struct {
	tenants []string
	failFor string
}

func (r *recordingEnsurer) Execute(ctx context.Context) error {
	id := tenant.IDFromContext(ctx)
	r.tenants = append(r.tenants, id)
	if id == r.failFor {
		return errors.New("boom")
	}
	return nil
}

func TestSeedMessageCenterDefaults_seedsEveryUnseededTenant(t *testing.T) {
	ensurer := &recordingEnsurer{failFor: "t-2"}
	uc := appmc.NewSeedMessageCenterDefaultsUseCase(newMockGormDB(t, 1), &stubSetupRepo{unseeded: []string{"t-1", "t-2", "t-3"}}, ensurer)

	err := uc.Execute(context.Background())

	assert.Equal(t, []string{"t-1", "t-2", "t-3"}, ensurer.tenants, "one failing tenant doesn't stop the others")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "t-2")
}

type listOnly struct{ out []mcDomain.MessageTemplate }

func (l listOnly) Execute(context.Context) ([]mcDomain.MessageTemplate, error) { return l.out, nil }

func TestListMessageTemplatesWithDefaults_seedsBeforeListing(t *testing.T) {
	ensurer := &recordingEnsurer{}
	uc := appmc.NewListMessageTemplatesWithDefaultsUseCase(ensurer, listOnly{out: []mcDomain.MessageTemplate{{ID: "a"}}})

	out, err := uc.Execute(tenant.AsTenant(context.Background(), "t-1"))

	require.NoError(t, err)
	assert.Equal(t, []string{"t-1"}, ensurer.tenants)
	assert.Len(t, out, 1)
}

func TestListMessageTemplatesWithDefaults_seedErrorStopsTheList(t *testing.T) {
	uc := appmc.NewListMessageTemplatesWithDefaultsUseCase(&recordingEnsurer{failFor: "t-1"}, listOnly{})

	_, err := uc.Execute(tenant.AsTenant(context.Background(), "t-1"))

	assert.Error(t, err)
}

func TestEnsureMessageCenterDefaults_withoutATenantDoesNothing(t *testing.T) {
	templates := &stubMessageTemplateRepo{}
	uc := appmc.NewEnsureMessageCenterDefaultsUseCase(newMockGormDB(t, 0), templates, &stubAutomationRuleRepo{}, &stubSetupRepo{}, &eventrecordtest.Outbox{}, dutchSettings())

	require.NoError(t, uc.Execute(tenant.WithSuperAdmin(context.Background())), "a cross-tenant super admin has no tenant to seed")
	assert.Empty(t, templates.created)
}
