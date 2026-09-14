package messagecenter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestUpdateAutomationRule_success(t *testing.T) {
	repo := &stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{
		"rule-1": {ID: "rule-1", Name: "Old", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1", Enabled: true},
	}}
	templateRepo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": {ID: "template-1"}}}
	uc := appmc.NewUpdateAutomationRuleUseCase(newMockGormDB(t, 1), repo, templateRepo)

	rule, err := uc.Execute(context.Background(), appmc.UpdateAutomationRuleInput{
		ID: "rule-1", Name: "New", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1", Enabled: false,
	})

	require.NoError(t, err)
	require.Len(t, repo.updated, 1)
	assert.Equal(t, "New", rule.Name)
	assert.False(t, rule.Enabled)
}

func TestUpdateAutomationRule_notFound(t *testing.T) {
	repo := &stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{}}
	uc := appmc.NewUpdateAutomationRuleUseCase(newMockGormDB(t, 1), repo, &stubMessageTemplateRepo{})

	_, err := uc.Execute(context.Background(), appmc.UpdateAutomationRuleInput{
		ID: "missing", Name: "New", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1",
	})

	assert.ErrorIs(t, err, mcDomain.ErrRuleNotFound)
}
