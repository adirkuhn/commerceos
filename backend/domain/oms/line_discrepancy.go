package oms

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrLineDiscrepancyNotFound = errors.New("order difference not found")
	ErrLineDiscrepancyNotOpen  = errors.New("order difference is already dismissed or resolved")
)

// LineDiscrepancyKind is how an imported order differs from its channel's
// own order.
type LineDiscrepancyKind string

const (
	// LineDiscrepancyMissing: the channel's order has an item this order
	// doesn't.
	LineDiscrepancyMissing LineDiscrepancyKind = "missing"
	// LineDiscrepancyQuantity: both have the item, in different quantities.
	LineDiscrepancyQuantity LineDiscrepancyKind = "quantity"
	// LineDiscrepancyTotal: the channel's own lines don't add up to its own
	// order total -- it was read while the channel was still saving it, so
	// items may be missing that can't be named.
	LineDiscrepancyTotal LineDiscrepancyKind = "total"
)

// totalItemKey is the single item key a LineDiscrepancyTotal is stored under.
const totalItemKey = "total"

// ChannelLine is one line of the channel's own order (e.g. a Woo line item).
type ChannelLine struct {
	SKUCode  string
	Name     string
	Quantity int64
}

// LineDiscrepancy is one difference between an imported order and the
// channel's order. It is only ever reported -- CommerceOS never changes an
// order on its own because of it. Staff decide: fix the order by hand
// (the difference then resolves itself on the next comparison) or dismiss
// it.
type LineDiscrepancy struct {
	ID           string
	TenantID     string
	SalesOrderID string
	Kind         LineDiscrepancyKind
	// ItemKey identifies the item being compared (normalised SKU code, or
	// name when there is none) -- one discrepancy per kind and item.
	ItemKey                string
	SKUCode                string
	Name                   string
	ChannelQuantity        int64
	LocalQuantity          int64
	ChannelTotalMinorUnits int64
	LocalTotalMinorUnits   int64
	DetectedAt             time.Time
	DismissedAt            *time.Time
	ResolvedAt             *time.Time
}

func (d LineDiscrepancy) IsOpen() bool { return d.DismissedAt == nil && d.ResolvedAt == nil }

// Dismiss marks an open difference as handled by staff.
func (d *LineDiscrepancy) Dismiss(now time.Time) error {
	if !d.IsOpen() {
		return ErrLineDiscrepancyNotOpen
	}
	d.DismissedAt = &now
	return nil
}

func (d LineDiscrepancy) sameNumbers(o LineDiscrepancy) bool {
	return d.ChannelQuantity == o.ChannelQuantity && d.LocalQuantity == o.LocalQuantity &&
		d.ChannelTotalMinorUnits == o.ChannelTotalMinorUnits && d.LocalTotalMinorUnits == o.LocalTotalMinorUnits
}

func itemKey(skuCode, name string) string {
	if sku := strings.ToLower(strings.TrimSpace(skuCode)); sku != "" {
		return "sku:" + sku
	}
	return "name:" + strings.ToLower(strings.TrimSpace(name))
}

// CompareChannelLines reports every channel item missing from the order or
// present in a different quantity, comparing per item (summed over lines).
// Items only on the order are not reported: extra items and substitutions
// made while packing are legitimate local changes.
func CompareChannelLines(channel []ChannelLine, local []SalesOrderLineItem) []LineDiscrepancy {
	localQty := map[string]int64{}
	for _, li := range local {
		localQty[itemKey(li.SKUCode, li.Name)] += li.Quantity
	}

	var order []string
	byKey := map[string]*LineDiscrepancy{}
	for _, cl := range channel {
		if cl.Quantity <= 0 {
			continue
		}
		key := itemKey(cl.SKUCode, cl.Name)
		d, ok := byKey[key]
		if !ok {
			d = &LineDiscrepancy{ItemKey: key, SKUCode: strings.TrimSpace(cl.SKUCode), Name: cl.Name}
			byKey[key] = d
			order = append(order, key)
		}
		d.ChannelQuantity += cl.Quantity
	}

	var out []LineDiscrepancy
	for _, key := range order {
		d := *byKey[key]
		d.LocalQuantity = localQty[key]
		switch {
		case d.LocalQuantity == 0:
			d.Kind = LineDiscrepancyMissing
		case d.LocalQuantity != d.ChannelQuantity:
			d.Kind = LineDiscrepancyQuantity
		default:
			continue
		}
		out = append(out, d)
	}
	return out
}

// totalToleranceMinorUnits absorbs per-line tax rounding: a channel's
// lines can add up to a cent or two off its own total.
const totalToleranceMinorUnits = 2

// ChannelTotalsDiffer reports whether a channel order's lines add up to
// something other than its own total -- beyond rounding.
func ChannelTotalsDiffer(channelTotal, linesTotal int64) bool {
	diff := channelTotal - linesTotal
	return diff > totalToleranceMinorUnits || diff < -totalToleranceMinorUnits
}

// TotalDiscrepancy reports that the channel's lines (and so the imported
// lines) add up to localTotal while the channel's order total is
// channelTotal.
func TotalDiscrepancy(channelTotal, localTotal int64) LineDiscrepancy {
	return LineDiscrepancy{
		Kind: LineDiscrepancyTotal, ItemKey: totalItemKey,
		ChannelTotalMinorUnits: channelTotal, LocalTotalMinorUnits: localTotal,
	}
}

// LineDiscrepancyPlan is what ReconcileLineDiscrepancies decided to store.
type LineDiscrepancyPlan struct {
	Create []LineDiscrepancy
	Update []LineDiscrepancy
}

// ReconcileLineDiscrepancies merges a fresh comparison (found) into an
// order's stored differences, for the kinds that comparison checked:
//   - new difference: created
//   - open, numbers changed: updated
//   - open, no longer found: resolved (the order was fixed)
//   - dismissed, same numbers: left dismissed; numbers changed: reopened
//   - resolved, found again: reopened
func ReconcileLineDiscrepancies(existing, found []LineDiscrepancy, kinds []LineDiscrepancyKind, now time.Time) LineDiscrepancyPlan {
	checked := map[LineDiscrepancyKind]bool{}
	for _, k := range kinds {
		checked[k] = true
	}
	key := func(d LineDiscrepancy) string { return string(d.Kind) + "|" + d.ItemKey }

	stored := map[string]LineDiscrepancy{}
	for _, d := range existing {
		if checked[d.Kind] {
			stored[key(d)] = d
		}
	}

	var plan LineDiscrepancyPlan
	seen := map[string]bool{}
	for _, f := range found {
		k := key(f)
		seen[k] = true
		s, ok := stored[k]
		if !ok {
			f.DetectedAt = now
			plan.Create = append(plan.Create, f)
			continue
		}
		unchanged := s.sameNumbers(f)
		switch {
		case s.IsOpen() && unchanged:
		case s.DismissedAt != nil && unchanged:
		default:
			plan.Update = append(plan.Update, reopened(s, f, now))
		}
	}
	for k, s := range stored {
		if !seen[k] && s.IsOpen() {
			s.ResolvedAt = &now
			plan.Update = append(plan.Update, s)
		}
	}
	return plan
}

func reopened(stored, found LineDiscrepancy, now time.Time) LineDiscrepancy {
	d := stored
	d.SKUCode, d.Name = found.SKUCode, found.Name
	d.ChannelQuantity, d.LocalQuantity = found.ChannelQuantity, found.LocalQuantity
	d.ChannelTotalMinorUnits, d.LocalTotalMinorUnits = found.ChannelTotalMinorUnits, found.LocalTotalMinorUnits
	if !stored.IsOpen() {
		d.DetectedAt = now
	}
	d.DismissedAt, d.ResolvedAt = nil, nil
	return d
}

// OpenLineDiscrepancy is an open difference with the order it belongs to,
// for the review list.
type OpenLineDiscrepancy struct {
	LineDiscrepancy
	Order SalesOrder
}

// LineDiscrepancyRepository stores LineDiscrepancies (tenant-scoped).
type LineDiscrepancyRepository interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]LineDiscrepancy, error)
	FindByID(ctx context.Context, id string) (*LineDiscrepancy, error)
	Create(ctx context.Context, d *LineDiscrepancy) error
	// Save persists quantities, totals, DetectedAt, DismissedAt, ResolvedAt.
	Save(ctx context.Context, d *LineDiscrepancy) error
	// ListOpen returns open differences with their orders, newest first.
	ListOpen(ctx context.Context, page, perPage int) ([]OpenLineDiscrepancy, int64, error)
}
