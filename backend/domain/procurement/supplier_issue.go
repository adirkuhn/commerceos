package procurement

import (
	"context"
	"time"
)

// SupplierIssueKind is what went wrong with a delivery, as told to the
// supplier.
type SupplierIssueKind string

const (
	// SupplierIssueMissing: fewer units arrived (across every completed
	// booking of the PO) than were ordered.
	SupplierIssueMissing SupplierIssueKind = "missing"
	SupplierIssueDamaged SupplierIssueKind = "damaged"
	SupplierIssueLost    SupplierIssueKind = "lost"
)

// SupplierIssueSource is where an issue was recorded; with SourceID it
// identifies the issue (see SupplierIssueRef).
type SupplierIssueSource string

const (
	// SupplierIssueSourceBookingShortfall: SourceID is the PO line item
	// that is short.
	SupplierIssueSourceBookingShortfall SupplierIssueSource = "booking_shortfall"
	// SupplierIssueSourceBookingScan: SourceID is a damaged/lost
	// BookingScan.
	SupplierIssueSourceBookingScan SupplierIssueSource = "booking_scan"
	// SupplierIssueSourceReceivingDiscrepancy: SourceID is a damaged_item
	// ReceivingDiscrepancy flagged on the PO's Receiving section.
	SupplierIssueSourceReceivingDiscrepancy SupplierIssueSource = "receiving_discrepancy"
)

func (s SupplierIssueSource) Valid() bool {
	switch s {
	case SupplierIssueSourceBookingShortfall, SupplierIssueSourceBookingScan, SupplierIssueSourceReceivingDiscrepancy:
		return true
	}
	return false
}

// SupplierIssueStatus tracks the claim with the supplier.
type SupplierIssueStatus string

const (
	SupplierIssueStatusOpen SupplierIssueStatus = "open"
	// SupplierIssueStatusClaimed: reported to the supplier, waiting on a
	// credit note or replacement.
	SupplierIssueStatusClaimed SupplierIssueStatus = "claimed"
	// SupplierIssueStatusResolved: the supplier settled it.
	SupplierIssueStatusResolved SupplierIssueStatus = "resolved"
)

func (s SupplierIssueStatus) Valid() bool {
	switch s {
	case SupplierIssueStatusOpen, SupplierIssueStatusClaimed, SupplierIssueStatusResolved:
		return true
	}
	return false
}

// SupplierIssueRef identifies one issue. Issues themselves are derived on
// demand from bookings and receiving flags, never stored -- only their
// claim status is (SupplierIssueState).
type SupplierIssueRef struct {
	Source   SupplierIssueSource
	SourceID string
}

// SupplierIssue is one line of the "what to send back to the supplier"
// list of a purchase order. Quantities are in stock units.
type SupplierIssue struct {
	Ref  SupplierIssueRef
	Kind SupplierIssueKind
	// LineItemID is the PO line the issue belongs to; nil when a scanned
	// SKU matches no line of the PO.
	LineItemID *string
	SKUID      *string
	// SupplierSKU and SupplierDescription are the line's text as the
	// supplier wrote it -- what the supplier recognises.
	SupplierSKU         string
	SupplierDescription string
	// ExpectedQty and ReceivedQty are set for missing issues only.
	ExpectedQty int64
	ReceivedQty int64
	Quantity    int64
	Reason      string
	ReportedBy  string
	OccurredAt  time.Time

	Status          SupplierIssueStatus
	StatusNote      string
	StatusUpdatedAt *time.Time
}

// SupplierIssueState is the stored claim status of one issue.
type SupplierIssueState struct {
	TenantID        string
	PurchaseOrderID string
	Ref             SupplierIssueRef
	Status          SupplierIssueStatus
	Note            string
	UpdatedBy       string
	UpdatedAt       time.Time
}

type SupplierIssueStateRepository interface {
	ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]SupplierIssueState, error)
	// Upsert creates or replaces the state of s.Ref.
	Upsert(ctx context.Context, s *SupplierIssueState) error
}

// Shortfall is a product line that received fewer stock units than it
// ordered.
type Shortfall struct {
	LineItem    PurchaseOrderLineItem
	ExpectedQty int64
	ReceivedQty int64
}

// Shortfalls compares every product line with the scans made for it. A
// damaged/lost write-off counts as arrived: the stock came, it just isn't
// sellable. Charge lines (shipping, discount, ...) can't be scanned and are
// skipped. Pass the scans of every completed booking of the PO -- a later
// delivery makes up for an earlier short one.
func Shortfalls(lineItems []PurchaseOrderLineItem, scans []BookingScan) []Shortfall {
	scannedBySKU := make(map[string]int64, len(scans))
	for _, s := range scans {
		scannedBySKU[s.SKUID] += s.Quantity + s.WriteOffQuantity
	}
	var out []Shortfall
	for _, li := range lineItems {
		if !li.LineType.IsProduct() {
			continue
		}
		var received int64
		if li.SKUID != nil {
			received = scannedBySKU[*li.SKUID]
		}
		// Scans are in stock units, li.Quantity in purchased items.
		expected := li.Quantity * li.EffectiveUnitsPerPurchasedItem()
		if received < expected {
			out = append(out, Shortfall{LineItem: li, ExpectedQty: expected, ReceivedQty: received})
		}
	}
	return out
}

// BookingWithScans is one completed booking with its scans.
type BookingWithScans struct {
	Booking Booking
	Scans   []BookingScan
}

type SupplierIssueInputs struct {
	LineItems []PurchaseOrderLineItem
	// Bookings are the PO's completed bookings only: an unfinished one can
	// still be corrected.
	Bookings               []BookingWithScans
	ReceivingDiscrepancies []ReceivingDiscrepancy
	States                 []SupplierIssueState
}

// CollectSupplierIssues lists everything wrong with a PO's deliveries:
// missing units, then damaged/lost scans, then damaged flags from the
// Receiving section. Nothing is missing before the first completed booking.
func CollectSupplierIssues(in SupplierIssueInputs) []SupplierIssue {
	states := make(map[SupplierIssueRef]SupplierIssueState, len(in.States))
	for _, s := range in.States {
		states[s.Ref] = s
	}
	lineByID := make(map[string]PurchaseOrderLineItem, len(in.LineItems))
	lineBySKU := make(map[string]PurchaseOrderLineItem, len(in.LineItems))
	for _, li := range in.LineItems {
		lineByID[li.ID] = li
		if li.SKUID != nil && li.LineType.IsProduct() {
			if _, seen := lineBySKU[*li.SKUID]; !seen {
				lineBySKU[*li.SKUID] = li
			}
		}
	}

	var issues []SupplierIssue
	if len(in.Bookings) > 0 {
		var scans []BookingScan
		var lastCompleted time.Time
		for _, b := range in.Bookings {
			scans = append(scans, b.Scans...)
			if b.Booking.CompletedAt != nil && b.Booking.CompletedAt.After(lastCompleted) {
				lastCompleted = *b.Booking.CompletedAt
			}
		}
		for _, sf := range Shortfalls(in.LineItems, scans) {
			issue := issueForLine(SupplierIssueRef{Source: SupplierIssueSourceBookingShortfall, SourceID: sf.LineItem.ID}, SupplierIssueMissing, &sf.LineItem)
			issue.ExpectedQty, issue.ReceivedQty, issue.Quantity = sf.ExpectedQty, sf.ReceivedQty, sf.ExpectedQty-sf.ReceivedQty
			issue.OccurredAt = lastCompleted
			issues = append(issues, issue)
		}
		for _, s := range scans {
			kind, ok := writeOffKind(s.ActionType)
			if !ok {
				continue
			}
			var line *PurchaseOrderLineItem
			if li, found := lineBySKU[s.SKUID]; found {
				line = &li
			}
			issue := issueForLine(SupplierIssueRef{Source: SupplierIssueSourceBookingScan, SourceID: s.ID}, kind, line)
			skuID := s.SKUID
			issue.SKUID = &skuID
			issue.Quantity, issue.Reason, issue.ReportedBy, issue.OccurredAt = s.WriteOffQuantity, s.Reason, s.ScannedBy, s.ScannedAt
			issues = append(issues, issue)
		}
	}
	for _, d := range in.ReceivingDiscrepancies {
		if d.Type != DiscrepancyTypeDamagedItem {
			continue
		}
		var line *PurchaseOrderLineItem
		if d.LineItemID != nil {
			if li, found := lineByID[*d.LineItemID]; found {
				line = &li
			}
		}
		issue := issueForLine(SupplierIssueRef{Source: SupplierIssueSourceReceivingDiscrepancy, SourceID: d.ID}, SupplierIssueDamaged, line)
		issue.Quantity, issue.Reason, issue.OccurredAt = d.ReceivedQty, d.Details, d.CreatedAt
		issues = append(issues, issue)
	}

	for i := range issues {
		issues[i].Status = SupplierIssueStatusOpen
		if st, ok := states[issues[i].Ref]; ok {
			updatedAt := st.UpdatedAt
			issues[i].Status, issues[i].StatusNote, issues[i].StatusUpdatedAt = st.Status, st.Note, &updatedAt
		}
	}
	return issues
}

// FindSupplierIssue returns the issue with ref.
func FindSupplierIssue(issues []SupplierIssue, ref SupplierIssueRef) (SupplierIssue, bool) {
	for _, issue := range issues {
		if issue.Ref == ref {
			return issue, true
		}
	}
	return SupplierIssue{}, false
}

func writeOffKind(a BookingScanActionType) (SupplierIssueKind, bool) {
	switch a {
	case BookingScanActionDamaged:
		return SupplierIssueDamaged, true
	case BookingScanActionLost:
		return SupplierIssueLost, true
	}
	return "", false
}

func issueForLine(ref SupplierIssueRef, kind SupplierIssueKind, line *PurchaseOrderLineItem) SupplierIssue {
	issue := SupplierIssue{Ref: ref, Kind: kind}
	if line != nil {
		id := line.ID
		issue.LineItemID, issue.SKUID = &id, line.SKUID
		issue.SupplierSKU, issue.SupplierDescription = line.RawSKU, line.RawDescription
	}
	return issue
}
