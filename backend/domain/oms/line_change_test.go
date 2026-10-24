package oms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/oms"
)


func TestMatchAddedLines_matchesEachExtraToTheLineItBecame(t *testing.T) {
	lines := []oms.SalesOrderLineItem{
		{ID: "line-orig", SKUID: strPtr("sku-a"), Name: "Guaraná", Quantity: 1},
		{ID: "line-extra-1", SKUID: strPtr("sku-a"), Name: "Guaraná", Quantity: 1},
		{ID: "line-extra-2", SKUID: strPtr("sku-a"), Name: "Guaraná", Quantity: 1},
	}
	changes := []oms.StagedChange{
		{ID: "c1", Action: oms.ChangeFreebie, SKUID: "sku-a", Quantity: 1, Reason: "Guaraná"},
		{ID: "c2", Action: oms.ChangeNewItem, SKUID: "sku-a", Quantity: 1, Reason: "Guaraná"},
		{ID: "c3", Action: oms.ChangeSubstitute, SKUID: "sku-a", Quantity: 1, Reason: "Guaraná", LineItemID: strPtr("line-orig")},
	}

	got := oms.MatchAddedLines(changes, lines)

	assert.Len(t, got, 2)
	assert.NotEqual(t, got["c1"], got["c2"])
	assert.NotContains(t, got, "c3")
}

func TestMatchAddedLines_leavesAnExtraThatNeverBecameALineUnmatched(t *testing.T) {
	lines := []oms.SalesOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-a"), Name: "Guaraná", Quantity: 2}}
	changes := []oms.StagedChange{{ID: "c1", Action: oms.ChangeFreebie, SKUID: "sku-a", Quantity: 1, Reason: "Guaraná"}}

	assert.Empty(t, oms.MatchAddedLines(changes, lines))
}
