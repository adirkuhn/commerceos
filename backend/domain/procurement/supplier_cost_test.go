package procurement_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/procurement"
)

var day = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

func on(d int) time.Time { return day.AddDate(0, 0, d) }

func sup(id string) *string { return &id }

func receipt(line, supplier string, at time.Time, qty, landed int64) procurement.SupplierCostEntry {
	e := procurement.SupplierCostEntry{
		SKUID: "sku-1", PurchaseOrderID: "po-" + line, PurchaseOrderLineItemID: line,
		Kind: procurement.SupplierCostKindReceipt, Quantity: qty, UnitCostMinorUnits: landed,
		LandedUnitCostMinorUnits: landed, CurrencyCode: "EUR", ReceivedAt: at,
	}
	if supplier != "" {
		e.SupplierID = sup(supplier)
	}
	return e
}

func withKind(e procurement.SupplierCostEntry, k procurement.SupplierCostKind, qty, landed int64, at time.Time) procurement.SupplierCostEntry {
	e.Kind, e.Quantity, e.LandedUnitCostMinorUnits, e.ReceivedAt = k, qty, landed, at
	return e
}

func TestSummarizeSupplierCosts_lastCostAndAveragesPerSupplierAndOverall(t *testing.T) {
	entries := []procurement.SupplierCostEntry{
		receipt("l1", "acme", on(0), 10, 500),
		receipt("l2", "bolt", on(1), 30, 300),
		receipt("l3", "acme", on(2), 10, 700),
	}

	s := procurement.SummarizeSupplierCosts(entries, "EUR")

	require.NotNil(t, s.Last)
	assert.Equal(t, int64(700), s.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, "l3", s.Last.PurchaseOrderLineItemID)
	assert.Equal(t, "acme", *s.Last.SupplierID)
	assert.Equal(t, on(2), s.Last.ReceivedAt)
	require.NotNil(t, s.AverageMinorUnits)
	assert.Equal(t, int64(420), *s.AverageMinorUnits, "(10*500 + 30*300 + 10*700) / 50, weighted by quantity")
	require.Len(t, s.BySupplier, 2)
	acme, bolt := s.BySupplier[0], s.BySupplier[1]
	assert.Equal(t, "acme", *acme.SupplierID, "most recent supplier first")
	assert.Equal(t, int64(600), acme.AverageMinorUnits)
	assert.Equal(t, int64(700), *acme.LastLandedUnitCostMinorUnits)
	assert.Equal(t, on(2), acme.LastReceivedAt)
	assert.Equal(t, int64(20), acme.Quantity)
	assert.Equal(t, 2, acme.Receipts)
	assert.Equal(t, int64(300), bolt.AverageMinorUnits)
	assert.Equal(t, 1, bolt.Receipts)
}

func TestSummarizeSupplierCosts_zeroCostCountsInAveragesButIsNeverTheLastCost(t *testing.T) {
	entries := []procurement.SupplierCostEntry{
		receipt("l1", "acme", on(0), 10, 600),
		receipt("l2", "acme", on(1), 10, 0),
	}

	s := procurement.SummarizeSupplierCosts(entries, "EUR")

	require.NotNil(t, s.Last)
	assert.Equal(t, int64(600), s.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, int64(300), *s.AverageMinorUnits)
	assert.Equal(t, int64(600), *s.BySupplier[0].LastLandedUnitCostMinorUnits)
	assert.Equal(t, int64(300), s.BySupplier[0].AverageMinorUnits)
}

func TestSummarizeSupplierCosts_aFullyReversedReceiptFallsBackToThePreviousOne(t *testing.T) {
	last := receipt("l2", "bolt", on(1), 10, 800)
	entries := []procurement.SupplierCostEntry{
		receipt("l1", "acme", on(0), 10, 500),
		last,
		withKind(last, procurement.SupplierCostKindReversal, -10, 800, on(2)),
	}

	s := procurement.SummarizeSupplierCosts(entries, "EUR")

	require.NotNil(t, s.Last)
	assert.Equal(t, int64(500), s.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, int64(500), *s.AverageMinorUnits)
	require.Len(t, s.BySupplier, 1, "a supplier left with nothing received drops out")
}

func TestSummarizeSupplierCosts_aPartialReversalKeepsTheLineAtItsCost(t *testing.T) {
	r := receipt("l1", "acme", on(0), 10, 500)
	entries := []procurement.SupplierCostEntry{r, withKind(r, procurement.SupplierCostKindReversal, -3, 500, on(1))}

	s := procurement.SummarizeSupplierCosts(entries, "EUR")

	assert.Equal(t, int64(500), s.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, int64(7), s.BySupplier[0].Quantity)
}

func TestSummarizeSupplierCosts_aCorrectionRepricesItsLineWithoutMakingItRecent(t *testing.T) {
	old := receipt("l1", "acme", on(0), 10, 500)
	entries := []procurement.SupplierCostEntry{
		old,
		receipt("l2", "bolt", on(1), 10, 900),
		// invoice for l1 corrected to 6.00: out at the old cost, in at the new
		withKind(old, procurement.SupplierCostKindCorrection, -10, 500, on(2)),
		withKind(old, procurement.SupplierCostKindCorrection, 10, 600, on(2)),
	}

	s := procurement.SummarizeSupplierCosts(entries, "EUR")

	assert.Equal(t, int64(900), s.Last.LandedUnitCostMinorUnits, "l2 is still the last receipt")
	assert.Equal(t, int64(750), *s.AverageMinorUnits)
	assert.Equal(t, int64(600), s.BySupplier[1].AverageMinorUnits)
}

func TestSummarizeSupplierCosts_aCorrectionOfTheLastReceiptMovesTheLastCost(t *testing.T) {
	last := receipt("l1", "acme", on(0), 10, 500)
	entries := []procurement.SupplierCostEntry{
		last,
		withKind(last, procurement.SupplierCostKindCorrection, -10, 500, on(1)),
		withKind(last, procurement.SupplierCostKindCorrection, 10, 550, on(1)),
	}

	s := procurement.SummarizeSupplierCosts(entries, "EUR")

	assert.Equal(t, int64(550), s.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, on(0), s.Last.ReceivedAt, "the receipt's own date, not the correction's")
}

func TestSummarizeSupplierCosts_otherCurrenciesAreLeftOutAndCounted(t *testing.T) {
	usd := receipt("l2", "acme", on(1), 10, 900)
	usd.CurrencyCode = "USD"

	s := procurement.SummarizeSupplierCosts([]procurement.SupplierCostEntry{receipt("l1", "acme", on(0), 10, 500), usd}, "EUR")

	assert.Equal(t, int64(500), s.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, int64(500), *s.AverageMinorUnits)
	assert.Equal(t, 1, s.ExcludedOtherCurrency)
}

func TestSummarizeSupplierCosts_receiptsWithoutASupplierGroupTogether(t *testing.T) {
	s := procurement.SummarizeSupplierCosts([]procurement.SupplierCostEntry{receipt("l1", "", on(0), 10, 500)}, "EUR")

	require.Len(t, s.BySupplier, 1)
	assert.Nil(t, s.BySupplier[0].SupplierID)
	assert.Nil(t, s.Last.SupplierID)
}

func TestSummarizeSupplierCosts_nothingReceived(t *testing.T) {
	s := procurement.SummarizeSupplierCosts(nil, "EUR")

	assert.Nil(t, s.Last)
	assert.Nil(t, s.AverageMinorUnits)
	assert.Empty(t, s.BySupplier)
}

func TestSummarizeSupplierCosts_onlyFreeGoodsHasAnAverageButNoLastCost(t *testing.T) {
	s := procurement.SummarizeSupplierCosts([]procurement.SupplierCostEntry{receipt("l1", "acme", on(0), 5, 0)}, "EUR")

	assert.Nil(t, s.Last)
	require.NotNil(t, s.AverageMinorUnits)
	assert.Equal(t, int64(0), *s.AverageMinorUnits)
	assert.Nil(t, s.BySupplier[0].LastLandedUnitCostMinorUnits)
}

func TestCorrectionEntries_repriceAReceivedLine(t *testing.T) {
	line := []procurement.SupplierCostEntry{receipt("l1", "acme", on(0), 10, 500)}

	out, ok := procurement.CorrectionEntries(line, 1000, on(3))

	require.True(t, ok)
	require.Len(t, out, 2)
	assert.Equal(t, procurement.SupplierCostKindCorrection, out[0].Kind)
	assert.Equal(t, int64(-10), out[0].Quantity)
	assert.Equal(t, int64(500), out[0].LandedUnitCostMinorUnits)
	assert.Equal(t, int64(10), out[1].Quantity)
	assert.Equal(t, int64(600), out[1].LandedUnitCostMinorUnits, "500 + 1000/10")
	assert.Equal(t, on(3), out[1].ReceivedAt)
	assert.Equal(t, "acme", *out[1].SupplierID)
}

func TestCorrectionEntries_nothingToCorrectOnAnEmptyLine(t *testing.T) {
	r := receipt("l1", "acme", on(0), 10, 500)
	_, ok := procurement.CorrectionEntries([]procurement.SupplierCostEntry{r, withKind(r, procurement.SupplierCostKindReversal, -10, 500, on(1))}, 1000, on(2))
	assert.False(t, ok)

	_, ok = procurement.CorrectionEntries(nil, 1000, on(2))
	assert.False(t, ok)
}

func TestReversalEntry_takesReceivedStockBackOffTheLineAtItsCost(t *testing.T) {
	line := []procurement.SupplierCostEntry{receipt("l1", "acme", on(0), 10, 500)}

	e, ok := procurement.ReversalEntry(line, 3, on(1))

	require.True(t, ok)
	assert.Equal(t, procurement.SupplierCostKindReversal, e.Kind)
	assert.Equal(t, int64(-3), e.Quantity)
	assert.Equal(t, int64(500), e.LandedUnitCostMinorUnits)
	assert.Equal(t, on(1), e.ReceivedAt)
	assert.Equal(t, "l1", e.PurchaseOrderLineItemID)
}

func TestReversalEntry_neverTakesBackMoreThanStands(t *testing.T) {
	line := []procurement.SupplierCostEntry{receipt("l1", "acme", on(0), 2, 500)}

	e, ok := procurement.ReversalEntry(line, 5, on(1))

	require.True(t, ok)
	assert.Equal(t, int64(-2), e.Quantity)

	_, ok = procurement.ReversalEntry(append(line, e), 1, on(2))
	assert.False(t, ok, "nothing left standing")
	_, ok = procurement.ReversalEntry(nil, 1, on(2))
	assert.False(t, ok)
	_, ok = procurement.ReversalEntry(line, 0, on(2))
	assert.False(t, ok)
}

func TestPageSupplierCosts_newestFirstFilteredAndPaged(t *testing.T) {
	entries := []procurement.SupplierCostEntry{
		receipt("l1", "acme", on(0), 1, 100),
		receipt("l2", "bolt", on(1), 1, 200),
		receipt("l3", "acme", on(2), 1, 300),
		receipt("l4", "acme", on(3), 1, 400),
	}

	page := procurement.PageSupplierCosts(entries, "acme", 0, 2)

	assert.Equal(t, 3, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "l4", page.Items[0].PurchaseOrderLineItemID)
	assert.Equal(t, "l3", page.Items[1].PurchaseOrderLineItemID)
	require.NotNil(t, page.NextCursor)
	assert.Equal(t, 2, *page.NextCursor)

	last := procurement.PageSupplierCosts(entries, "acme", 2, 2)
	require.Len(t, last.Items, 1)
	assert.Equal(t, "l1", last.Items[0].PurchaseOrderLineItemID)
	assert.Nil(t, last.NextCursor)

	all := procurement.PageSupplierCosts(entries, "", 0, 10)
	assert.Equal(t, 4, all.Total)
	assert.Empty(t, procurement.PageSupplierCosts(entries, "", 9, 10).Items, "past the end")
}
