package procurement

import (
	"context"
	"sort"
	"time"
)

// SupplierCostKind says why a SupplierCostEntry exists.
type SupplierCostKind string

const (
	// SupplierCostKindReceipt: stock was booked in against a purchase
	// order line (procurement.goods.received).
	SupplierCostKindReceipt SupplierCostKind = "receipt"
	// SupplierCostKindReversal: received stock was taken back off a line
	// (e.g. a scanner booking correction) -- negative quantity.
	SupplierCostKindReversal SupplierCostKind = "reversal"
	// SupplierCostKindCorrection: a line's price was corrected after the
	// fact (e.g. a corrected supplier invoice) -- always a pair: the line's
	// quantity out at the old cost, back in at the new one.
	SupplierCostKindCorrection SupplierCostKind = "correction"
)

// SupplierCostEntry is one append-only row of a SKU's cost history by
// supplier (COM-11). History is never edited: a reversal or a correction
// adds rows. Quantity is signed; a line's cost is the value of its rows
// over their net quantity.
type SupplierCostEntry struct {
	ID                      string
	TenantID                string
	SKUID                   string
	SupplierID              *string // nil when the purchase order has no supplier
	PurchaseOrderID         string
	PurchaseOrderLineItemID string
	BatchID                 *string
	Kind                    SupplierCostKind
	Quantity                int64
	// UnitCostMinorUnits is the PO line's own price per unit;
	// LandedUnitCostMinorUnits adds its share of the order's shipping and
	// fees (application/procurement.AllocateLandedUnitCost).
	UnitCostMinorUnits       int64
	LandedUnitCostMinorUnits int64
	CurrencyCode             string
	ReceivedAt               time.Time
	// SourceKey makes writing idempotent: the same source (e.g. a
	// goods-received event) never adds a second row.
	SourceKey string
	CreatedAt time.Time
}

// SupplierCostRepository is the append-only store of SupplierCostEntry.
type SupplierCostRepository interface {
	// Append adds entries; one whose SourceKey is already stored is
	// skipped (not an error).
	Append(ctx context.Context, entries []SupplierCostEntry) error
	ListBySKU(ctx context.Context, skuID string) ([]SupplierCostEntry, error)
	ListByLine(ctx context.Context, lineItemID string) ([]SupplierCostEntry, error)
}

// LastSupplierCost is the receipt a SKU's cost price comes from.
type LastSupplierCost struct {
	LandedUnitCostMinorUnits int64
	PurchaseOrderID          string
	PurchaseOrderLineItemID  string
	SupplierID               *string
	ReceivedAt               time.Time
}

// SupplierCostStat is one supplier's line in the summary.
type SupplierCostStat struct {
	SupplierID *string
	// LastLandedUnitCostMinorUnits is this supplier's last paid receipt;
	// nil when everything it sent was free.
	LastLandedUnitCostMinorUnits *int64
	LastReceivedAt               time.Time
	AverageMinorUnits            int64
	Quantity                     int64
	Receipts                     int
}

// SupplierCostSummary is what a SKU's history says.
type SupplierCostSummary struct {
	// Last is the most recent paid receipt still standing; nil when none.
	Last *LastSupplierCost
	// AverageMinorUnits is the average landed cost over everything ever
	// received from every supplier, weighted by quantity; nil when nothing
	// is received.
	AverageMinorUnits *int64
	// BySupplier: most recently received from first.
	BySupplier []SupplierCostStat
	// ExcludedOtherCurrency counts rows left out for not being in the
	// summary's currency.
	ExcludedOtherCurrency int
}

// supplierLine is one purchase order line, its rows netted.
type supplierLine struct {
	first      SupplierCostEntry
	receivedAt time.Time
	qty        int64
	value      int64
}

func (l supplierLine) standing() bool { return l.qty > 0 }

func (l supplierLine) cost() int64 { return roundDiv(l.value, l.qty) }

// SummarizeSupplierCosts nets each purchase order line's rows (receipts,
// reversals, corrections) and summarizes the lines still standing:
//
//   - Last: the most recently received line with a cost above zero -- a
//     free receipt never becomes the last cost, a fully reversed one drops
//     out, a corrected one keeps its receipt date at its corrected cost;
//   - averages weighted by quantity, free receipts included;
//   - rows not in currency are left out (and counted).
func SummarizeSupplierCosts(entries []SupplierCostEntry, currency string) SupplierCostSummary {
	var out SupplierCostSummary
	lines := netLines(entries, currency, &out.ExcludedOtherCurrency)

	var qty, value int64
	bySupplier := map[string]*SupplierCostStat{}
	var order []string
	for _, l := range lines {
		if !l.standing() {
			continue
		}
		qty += l.qty
		value += l.value
		if l.value > 0 && (out.Last == nil || l.receivedAt.After(out.Last.ReceivedAt)) {
			out.Last = &LastSupplierCost{
				LandedUnitCostMinorUnits: l.cost(), PurchaseOrderID: l.first.PurchaseOrderID,
				PurchaseOrderLineItemID: l.first.PurchaseOrderLineItemID, SupplierID: l.first.SupplierID,
				ReceivedAt: l.receivedAt,
			}
		}
		key := supplierKey(l.first.SupplierID)
		st, ok := bySupplier[key]
		if !ok {
			st = &SupplierCostStat{SupplierID: l.first.SupplierID}
			bySupplier[key] = st
			order = append(order, key)
		}
		st.Quantity += l.qty
		st.AverageMinorUnits += l.value // summed value until divided below
		st.Receipts++
		if l.receivedAt.After(st.LastReceivedAt) {
			st.LastReceivedAt = l.receivedAt
		}
		if l.value > 0 {
			c := l.cost()
			st.LastLandedUnitCostMinorUnits = &c
		}
	}
	if qty > 0 {
		avg := roundDiv(value, qty)
		out.AverageMinorUnits = &avg
	}
	for _, key := range order {
		st := bySupplier[key]
		st.AverageMinorUnits = roundDiv(st.AverageMinorUnits, st.Quantity)
		out.BySupplier = append(out.BySupplier, *st)
	}
	sort.SliceStable(out.BySupplier, func(i, j int) bool {
		return out.BySupplier[i].LastReceivedAt.After(out.BySupplier[j].LastReceivedAt)
	})
	return out
}

// netLines groups rows in currency by purchase order line, ordered by when
// each line was received (its first receipt; corrections and reversals
// never make a line more recent).
func netLines(entries []SupplierCostEntry, currency string, excluded *int) []supplierLine {
	idx := map[string]int{}
	var lines []supplierLine
	for _, e := range entries {
		if e.CurrencyCode != currency {
			*excluded++
			continue
		}
		i, ok := idx[e.PurchaseOrderLineItemID]
		if !ok {
			i = len(lines)
			idx[e.PurchaseOrderLineItemID] = i
			lines = append(lines, supplierLine{first: e})
		}
		l := &lines[i]
		l.qty += e.Quantity
		l.value += e.Quantity * e.LandedUnitCostMinorUnits
		if e.Kind == SupplierCostKindReceipt && (l.receivedAt.IsZero() || e.ReceivedAt.After(l.receivedAt)) {
			l.receivedAt = e.ReceivedAt
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].receivedAt.Before(lines[j].receivedAt) })
	return lines
}

// CorrectionEntries reprices one purchase order line after the fact: its
// standing quantity out at its current cost, back in at that cost plus
// adjustmentValueMinorUnits spread over it. lineEntries are that line's
// rows. ok is false when nothing of the line is standing.
func CorrectionEntries(lineEntries []SupplierCostEntry, adjustmentValueMinorUnits int64, at time.Time) ([]SupplierCostEntry, bool) {
	if len(lineEntries) == 0 {
		return nil, false
	}
	var excluded int
	lines := netLines(lineEntries, lineEntries[0].CurrencyCode, &excluded)
	if len(lines) != 1 || !lines[0].standing() {
		return nil, false
	}
	l := lines[0]
	oldCost := l.cost()
	newCost := oldCost + roundDiv(adjustmentValueMinorUnits, l.qty)
	out, in := l.first, l.first
	out.ID, in.ID = "", ""
	out.Kind, in.Kind = SupplierCostKindCorrection, SupplierCostKindCorrection
	out.Quantity, in.Quantity = -l.qty, l.qty
	out.LandedUnitCostMinorUnits, in.LandedUnitCostMinorUnits = oldCost, newCost
	out.ReceivedAt, in.ReceivedAt = at, at
	out.BatchID, in.BatchID = nil, nil
	return []SupplierCostEntry{out, in}, true
}

func supplierKey(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

// roundDiv divides rounding half away from zero (smallest currency unit
// convention used for money across the codebase).
func roundDiv(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	if (a < 0) != (b < 0) {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}
