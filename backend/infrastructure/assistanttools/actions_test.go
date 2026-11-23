package assistanttools_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/assistanttools"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

func TestProposalActions_everyKindAndToolOnce(t *testing.T) {
	db := &gorm.DB{}
	as := assistanttools.ProposalActions(db, repositories.NewOutboxRepository(db))

	kinds, tools := map[assistant.ProposalKind]bool{}, map[string]bool{}
	for _, a := range as {
		assert.False(t, kinds[a.Kind()], "duplicate kind %q", a.Kind())
		assert.False(t, tools[a.Tool().Name], "duplicate tool %q", a.Tool().Name)
		kinds[a.Kind()], tools[a.Tool().Name] = true, true
	}
	assert.Equal(t, map[assistant.ProposalKind]bool{assistant.KindCreateQuote: true, assistant.KindConvertQuote: true}, kinds)
	assert.Equal(t, map[string]bool{"propose_quote": true, "propose_quote_conversion": true}, tools)

	read := map[string]bool{}
	for _, tool := range assistanttools.ReadTools(db) {
		read[tool.Definition().Name] = true
	}
	for name := range tools {
		assert.False(t, read[name], "proposal tool %q clashes with a read tool", name)
	}
}
