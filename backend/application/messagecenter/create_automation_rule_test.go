package messagecenter_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestCreateAutomationRule_success(t *testing.T) {
	repo := &stubAutomationRuleRepo{}
	templateRepo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": {ID: "template-1"}}}
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 1), repo, templateRepo, &eventrecordtest.Outbox{})

	rule, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "Notify on collect point delivery", TriggerSubject: "shipping.shipment.fulfillment_signal",
		Conditions: []mcDomain.Condition{{Field: "status_code", Operator: mcDomain.OperatorIn, Value: []any{float64(8), float64(19)}}},
		TemplateID: "template-1",
	})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.True(t, rule.Enabled)
}

func TestCreateAutomationRule_rejectsEmptyName(t *testing.T) {
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1",
	})

	assert.ErrorIs(t, err, mcDomain.ErrRuleNameRequired)
}

func TestCreateAutomationRule_rejectsInvalidTriggerSubject(t *testing.T) {
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "X", TriggerSubject: "bogus.subject", TemplateID: "template-1",
	})

	assert.ErrorIs(t, err, mcDomain.ErrInvalidTriggerSubject)
}

func TestCreateAutomationRule_rejectsMissingTemplateID(t *testing.T) {
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "X", TriggerSubject: "shipping.shipment.fulfillment_signal",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateIDRequired)
}

func TestCreateAutomationRule_rejectsInvalidCondition_emptyField(t *testing.T) {
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "X", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1",
		Conditions: []mcDomain.Condition{{Field: "", Operator: mcDomain.OperatorEq, Value: "x"}},
	})

	assert.ErrorIs(t, err, mcDomain.ErrInvalidCondition)
}

func TestCreateAutomationRule_rejectsInvalidCondition_badOperator(t *testing.T) {
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "X", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1",
		Conditions: []mcDomain.Condition{{Field: "status_code", Operator: mcDomain.Operator("bogus"), Value: "x"}},
	})

	assert.ErrorIs(t, err, mcDomain.ErrInvalidCondition)
}

func TestCreateAutomationRule_rejectsInOperatorWithNonArrayValue(t *testing.T) {
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 0), &stubAutomationRuleRepo{}, &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "X", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1",
		Conditions: []mcDomain.Condition{{Field: "status_code", Operator: mcDomain.OperatorIn, Value: "not-an-array"}},
	})

	assert.ErrorIs(t, err, mcDomain.ErrInvalidCondition)
}

func TestCreateAutomationRule_propagatesTemplateNotFound(t *testing.T) {
	repo := &stubAutomationRuleRepo{}
	templateRepo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{}}
	uc := appmc.NewCreateAutomationRuleUseCase(newMockGormDB(t, 1), repo, templateRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateAutomationRuleInput{
		Name: "X", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "missing-template",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNotFound)
}
