package messagecenter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestListAutomationRules_returnsEveryRule(t *testing.T) {
	repo := &stubAutomationRuleRepo{all: []mcDomain.AutomationRule{{ID: "rule-1"}, {ID: "rule-2"}}}
	uc := appmc.NewListAutomationRulesUseCase(newMockGormDB(t, 1), repo)

	rules, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, rules, 2)
}
