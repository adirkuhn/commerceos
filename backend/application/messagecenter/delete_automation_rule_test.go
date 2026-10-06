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

func TestDeleteAutomationRule_success(t *testing.T) {
	repo := &stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{"rule-1": {ID: "rule-1"}}}
	uc := appmc.NewDeleteAutomationRuleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "rule-1")

	require.NoError(t, err)
	assert.Equal(t, "rule-1", repo.deletedID)
}

func TestDeleteAutomationRule_notFound(t *testing.T) {
	repo := &stubAutomationRuleRepo{byID: map[string]mcDomain.AutomationRule{}}
	uc := appmc.NewDeleteAutomationRuleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, mcDomain.ErrRuleNotFound)
}
