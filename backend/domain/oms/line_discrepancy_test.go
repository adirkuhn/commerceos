package oms_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

var discrepancyNow = time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)

func localLine(sku, name string, qty int64) omsDomain.SalesOrderLineItem {
	return omsDomain.SalesOrderLineItem{SKUCode: sku, Name: name, Quantity: qty}
}

func TestCompareChannelLines_missingLine(t *testing.T) {
	channel := []omsDomain.ChannelLine{
		{SKUCode: "7891095031139", Name: "Batata palha", Quantity: 2},
		{SKUCode: "7896015976606", Name: "Mistura de Pão de Queijo - Pinduca - 500g", Quantity: 1},
	}
	local := []omsDomain.SalesOrderLineItem{localLine("7891095031139", "Batata palha", 2)}

	got := omsDomain.CompareChannelLines(channel, local)

	require.Len(t, got, 1)
	assert.Equal(t, omsDomain.LineDiscrepancyMissing, got[0].Kind)
	assert.Equal(t, "7896015976606", got[0].SKUCode)
	assert.Equal(t, "Mistura de Pão de Queijo - Pinduca - 500g", got[0].Name)
	assert.Equal(t, int64(1), got[0].ChannelQuantity)
	assert.Equal(t, int64(0), got[0].LocalQuantity)
}

func TestCompareChannelLines_quantityDiffers(t *testing.T) {
	got := omsDomain.CompareChannelLines(
		[]omsDomain.ChannelLine{{SKUCode: "A", Name: "Apple", Quantity: 3}},
		[]omsDomain.SalesOrderLineItem{localLine("A", "Apple", 2)},
	)

	require.Len(t, got, 1)
	assert.Equal(t, omsDomain.LineDiscrepancyQuantity, got[0].Kind)
	assert.Equal(t, int64(3), got[0].ChannelQuantity)
	assert.Equal(t, int64(2), got[0].LocalQuantity)
}

func TestCompareChannelLines_identicalOrder_noDiscrepancy(t *testing.T) {
	got := omsDomain.CompareChannelLines(
		[]omsDomain.ChannelLine{{SKUCode: "A", Name: "Apple", Quantity: 1}, {Name: "Gift wrap", Quantity: 1}},
		[]omsDomain.SalesOrderLineItem{localLine("A", "Apple", 1), localLine("", "Gift wrap", 1)},
	)

	assert.Empty(t, got)
}

func TestCompareChannelLines_sameSKUOnSeveralLines_comparesTheSum(t *testing.T) {
	got := omsDomain.CompareChannelLines(
		[]omsDomain.ChannelLine{{SKUCode: "A", Name: "Apple", Quantity: 1}, {SKUCode: "A", Name: "Apple", Quantity: 2}},
		[]omsDomain.SalesOrderLineItem{localLine("A", "Apple", 3)},
	)

	assert.Empty(t, got)
}

func TestCompareChannelLines_matchesSKUCaseAndSpaceInsensitively(t *testing.T) {
	got := omsDomain.CompareChannelLines(
		[]omsDomain.ChannelLine{{SKUCode: " par_tm_32_70g ", Name: "Bananinha", Quantity: 1}},
		[]omsDomain.SalesOrderLineItem{localLine("PAR_TM_32_70G", "Bananinha Paraibuna", 1)},
	)

	assert.Empty(t, got)
}

func TestCompareChannelLines_linesOnlyInCommerceOS_areIgnored(t *testing.T) {
	got := omsDomain.CompareChannelLines(
		[]omsDomain.ChannelLine{{SKUCode: "A", Name: "Apple", Quantity: 1}},
		[]omsDomain.SalesOrderLineItem{localLine("A", "Apple", 1), localLine("B", "Extra added at packing", 1)},
	)

	assert.Empty(t, got, "extra items added while packing are legitimate local changes")
}

func TestCompareChannelLines_ignoresNonPositiveChannelLines(t *testing.T) {
	got := omsDomain.CompareChannelLines(
		[]omsDomain.ChannelLine{{SKUCode: "A", Name: "Refund adjustment", Quantity: -1}},
		nil,
	)

	assert.Empty(t, got, "the import skips these too")
}

func TestTotalDiscrepancy(t *testing.T) {
	d := omsDomain.TotalDiscrepancy(3641, 3293)

	assert.Equal(t, omsDomain.LineDiscrepancyTotal, d.Kind)
	assert.Equal(t, int64(3641), d.ChannelTotalMinorUnits)
	assert.Equal(t, int64(3293), d.LocalTotalMinorUnits)
}

// ── ReconcileLineDiscrepancies ────────────────────────────────────────────────

func missing(sku string, channelQty int64) omsDomain.LineDiscrepancy {
	return omsDomain.CompareChannelLines([]omsDomain.ChannelLine{{SKUCode: sku, Name: sku, Quantity: channelQty}}, nil)[0]
}

var lineKinds = []omsDomain.LineDiscrepancyKind{omsDomain.LineDiscrepancyMissing, omsDomain.LineDiscrepancyQuantity}

func TestReconcile_newDiscrepancy_isCreated(t *testing.T) {
	plan := omsDomain.ReconcileLineDiscrepancies(nil, []omsDomain.LineDiscrepancy{missing("A", 1)}, lineKinds, discrepancyNow)

	require.Len(t, plan.Create, 1)
	assert.Equal(t, discrepancyNow, plan.Create[0].DetectedAt)
	assert.Empty(t, plan.Update)
}

func TestReconcile_stillOpenAndUnchanged_doesNothing(t *testing.T) {
	existing := missing("A", 1)
	existing.ID = "d-1"

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{existing}, []omsDomain.LineDiscrepancy{missing("A", 1)}, lineKinds, discrepancyNow)

	assert.Empty(t, plan.Create)
	assert.Empty(t, plan.Update)
}

func TestReconcile_openButNumbersChanged_isUpdated(t *testing.T) {
	existing := missing("A", 1)
	existing.ID = "d-1"

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{existing}, []omsDomain.LineDiscrepancy{missing("A", 2)}, lineKinds, discrepancyNow)

	require.Len(t, plan.Update, 1)
	assert.Equal(t, "d-1", plan.Update[0].ID)
	assert.Equal(t, int64(2), plan.Update[0].ChannelQuantity)
}

func TestReconcile_goneFromComparison_isResolved(t *testing.T) {
	existing := missing("A", 1)
	existing.ID = "d-1"

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{existing}, nil, lineKinds, discrepancyNow)

	require.Len(t, plan.Update, 1)
	require.NotNil(t, plan.Update[0].ResolvedAt)
	assert.Equal(t, discrepancyNow, *plan.Update[0].ResolvedAt)
}

func TestReconcile_dismissedAndUnchanged_staysDismissed(t *testing.T) {
	existing := missing("A", 1)
	existing.ID = "d-1"
	existing.DismissedAt = &discrepancyNow

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{existing}, []omsDomain.LineDiscrepancy{missing("A", 1)}, lineKinds, discrepancyNow)

	assert.Empty(t, plan.Create)
	assert.Empty(t, plan.Update)
}

func TestReconcile_dismissedButNumbersChanged_reopens(t *testing.T) {
	existing := missing("A", 1)
	existing.ID = "d-1"
	existing.DismissedAt = &discrepancyNow

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{existing}, []omsDomain.LineDiscrepancy{missing("A", 3)}, lineKinds, discrepancyNow)

	require.Len(t, plan.Update, 1)
	assert.Nil(t, plan.Update[0].DismissedAt)
	assert.Equal(t, int64(3), plan.Update[0].ChannelQuantity)
}

func TestReconcile_resolvedThenBackAgain_reopens(t *testing.T) {
	existing := missing("A", 1)
	existing.ID = "d-1"
	existing.ResolvedAt = &discrepancyNow

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{existing}, []omsDomain.LineDiscrepancy{missing("A", 1)}, lineKinds, discrepancyNow)

	require.Len(t, plan.Update, 1)
	assert.Nil(t, plan.Update[0].ResolvedAt)
}

func TestReconcile_onlyTouchesTheKindsChecked(t *testing.T) {
	total := omsDomain.TotalDiscrepancy(3641, 3293)
	total.ID = "d-total"

	plan := omsDomain.ReconcileLineDiscrepancies([]omsDomain.LineDiscrepancy{total}, nil, lineKinds, discrepancyNow)

	assert.Empty(t, plan.Update, "a line comparison says nothing about the total check")
}

func TestLineDiscrepancy_dismiss(t *testing.T) {
	d := missing("A", 1)

	require.NoError(t, d.Dismiss(discrepancyNow))
	require.NotNil(t, d.DismissedAt)
	assert.ErrorIs(t, d.Dismiss(discrepancyNow), omsDomain.ErrLineDiscrepancyNotOpen)
}

func TestChannelTotalsDiffer(t *testing.T) {
	assert.False(t, omsDomain.ChannelTotalsDiffer(3641, 3641))
	assert.False(t, omsDomain.ChannelTotalsDiffer(3641, 3639), "rounding")
	assert.False(t, omsDomain.ChannelTotalsDiffer(3641, 3643), "rounding")
	assert.True(t, omsDomain.ChannelTotalsDiffer(3641, 3294))
	assert.True(t, omsDomain.ChannelTotalsDiffer(3641, 3644))
}
