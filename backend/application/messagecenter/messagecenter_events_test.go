package messagecenter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func templateRepoWith(tmpl mcDomain.MessageTemplate) *stubMessageTemplateRepo {
	return &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{tmpl.ID: tmpl}}
}

func TestCreateMessageTemplate_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	_, err := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 1), &stubMessageTemplateRepo{}, outbox).Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "Shipped", Category: mcDomain.CategoryOrderStatus, Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "Hi",
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "messagecenter.message_template.created", e.Subject)
	assert.Equal(t, "template-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Shipped", "category": string(mcDomain.CategoryOrderStatus), "channels": []any{"email"}}, e.Data())
}

func TestUpdateMessageTemplate_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := templateRepoWith(mcDomain.MessageTemplate{ID: "t-1", Name: "Shipped", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "Hi", Enabled: true})

	_, err := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 1), repo, outbox).Execute(context.Background(), appmc.UpdateMessageTemplateInput{
		ID: "t-1", Name: "Shipped", Category: mcDomain.CategoryOrderStatus, Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "Hello", Enabled: false,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "messagecenter.message_template.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"body": {"Hi", "Hello"}, "enabled": {true, false}}, e.Changes())
}

func TestDeleteMessageTemplate_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	require.NoError(t, appmc.NewDeleteMessageTemplateUseCase(newMockGormDB(t, 1), templateRepoWith(mcDomain.MessageTemplate{ID: "t-1", Name: "Shipped"}), outbox).
		Execute(context.Background(), "t-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "messagecenter.message_template.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Shipped"}, e.Data())
}

func TestCreateAutomationRule_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	subject := mcDomain.KnownTriggerSubjects()[0]

	_, err := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 1), &stubAutomationRuleRepo{}, templateRepoWith(mcDomain.MessageTemplate{ID: "t-1"}), outbox).
		Execute(context.Background(), appmc.CreateAutomationRuleInput{Name: "Notify", TriggerSubject: subject, TemplateID: "t-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "messagecenter.automation_rule.created", e.Subject)
	assert.Equal(t, "rule-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Notify", "trigger_subject": subject, "template_id": "t-1"}, e.Data())
}

func TestUpdateAutomationRule_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	subject := mcDomain.KnownTriggerSubjects()[0]
	rules := &stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{"r-1": {ID: "r-1", Name: "Notify", TriggerSubject: subject, TemplateID: "t-1", Enabled: true}}}

	_, err := appmc.NewUpdateAutomationRuleUseCase(newMockGormDB(t, 1), rules, templateRepoWith(mcDomain.MessageTemplate{ID: "t-1"}), outbox).
		Execute(context.Background(), appmc.UpdateAutomationRuleInput{ID: "r-1", Name: "Notify customer", TriggerSubject: subject, TemplateID: "t-1", Enabled: true})

	require.NoError(t, err)
	assert.Equal(t, map[string][2]any{"name": {"Notify", "Notify customer"}}, onlyEvent(t, outbox).Changes())
}

func TestDeleteAutomationRule_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	rules := &stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{"r-1": {ID: "r-1", Name: "Notify"}}}

	require.NoError(t, appmc.NewDeleteAutomationRuleUseCase(newMockGormDB(t, 1), rules, outbox).Execute(context.Background(), "r-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "messagecenter.automation_rule.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Notify"}, e.Data())
}

func TestMessageCenterUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}

	_, err := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 1), &stubMessageTemplateRepo{}, failing).Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "x", Category: mcDomain.CategoryOrderStatus, Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "b",
	})
	assert.ErrorIs(t, err, boom)
	assert.ErrorIs(t, appmc.NewDeleteAutomationRuleUseCase(newMockGormDB(t, 1),
		&stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{"r-1": {ID: "r-1"}}}, failing).Execute(context.Background(), "r-1"), boom)
}
