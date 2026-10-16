package procurement_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/procurement"
)

func strPtr(s string) *string { return &s }

func productLine(id, skuID string, qty int64) procurement.PurchaseOrderLineItem {
	return procurement.PurchaseOrderLineItem{
		ID: id, SKUID: strPtr(skuID), Quantity: qty, LineType: procurement.LineItemTypeProduct,
		RawSKU: "SUP-" + id, RawDescription: "Supplier text " + id,
	}
}

func receiveScan(id, skuID string, qty int64) procurement.BookingScan {
	return procurement.BookingScan{ID: id, SKUID: skuID, ActionType: procurement.BookingScanActionReceive, Quantity: qty}
}

func TestShortfalls_sumsScansAcrossEveryBookingOfThePurchaseOrder(t *testing.T) {
	// First delivery brought 4 of 10, a later delivery the other 6: nothing
	// is missing any more, even though the first booking alone was short.
	lines := []procurement.PurchaseOrderLineItem{productLine("line-1", "sku-1", 10)}
	scans := []procurement.BookingScan{receiveScan("scan-1", "sku-1", 4), receiveScan("scan-2", "sku-1", 6)}

	require.Empty(t, procurement.Shortfalls(lines, scans))
}

func TestShortfalls_countsWriteOffsAsArrivedAndScalesByUnitsPerPurchasedItem(t *testing.T) {
	line := productLine("line-1", "sku-1", 2)
	line.UnitsPerPurchasedItem = 12
	scans := []procurement.BookingScan{
		receiveScan("scan-1", "sku-1", 10),
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurement.BookingScanActionDamaged, WriteOffQuantity: 3},
	}

	got := procurement.Shortfalls([]procurement.PurchaseOrderLineItem{line}, scans)

	require.Equal(t, []procurement.Shortfall{{LineItem: line, ExpectedQty: 24, ReceivedQty: 13}}, got)
}

func TestShortfalls_skipsChargeLines(t *testing.T) {
	shipping := procurement.PurchaseOrderLineItem{ID: "ship", Quantity: 1, LineType: procurement.LineItemTypeShipping}

	require.Empty(t, procurement.Shortfalls([]procurement.PurchaseOrderLineItem{shipping}, nil))
}

func TestCollectSupplierIssues_listsMissingDamagedAndLostWithTheirStatus(t *testing.T) {
	scannedAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	flaggedAt := scannedAt.Add(time.Hour)
	completedAt := scannedAt.Add(2 * time.Hour)
	updatedAt := completedAt.Add(time.Hour)
	lines := []procurement.PurchaseOrderLineItem{productLine("line-1", "sku-1", 10), productLine("line-2", "sku-2", 5)}
	in := procurement.SupplierIssueInputs{
		LineItems: lines,
		Bookings: []procurement.BookingWithScans{{
			Booking: procurement.Booking{ID: "booking-1", CompletedAt: &completedAt},
			Scans: []procurement.BookingScan{
				receiveScan("scan-1", "sku-1", 7),
				receiveScan("scan-2", "sku-2", 5),
				{ID: "scan-3", SKUID: "sku-2", ActionType: procurement.BookingScanActionDamaged, WriteOffQuantity: 2,
					Reason: "crushed box", ScannedAt: scannedAt, ScannedBy: "ana"},
				{ID: "scan-4", SKUID: "sku-1", ActionType: procurement.BookingScanActionLost, WriteOffQuantity: 1, ScannedAt: scannedAt},
			},
		}},
		ReceivingDiscrepancies: []procurement.ReceivingDiscrepancy{
			{ID: "disc-1", LineItemID: strPtr("line-2"), Type: procurement.DiscrepancyTypeDamagedItem, ReceivedQty: 1, Details: "leaking", CreatedAt: flaggedAt},
			{ID: "disc-2", LineItemID: strPtr("line-1"), Type: procurement.DiscrepancyTypeQuantityMismatch, ExpectedQty: 10, ReceivedQty: 8},
		},
		States: []procurement.SupplierIssueState{{
			Ref:    procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceBookingScan, SourceID: "scan-3"},
			Status: procurement.SupplierIssueStatusClaimed, Note: "emailed", UpdatedAt: updatedAt,
		}},
	}

	got := procurement.CollectSupplierIssues(in)

	require.Len(t, got, 4)
	missing := got[0]
	require.Equal(t, procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceBookingShortfall, SourceID: "line-1"}, missing.Ref)
	require.Equal(t, procurement.SupplierIssueMissing, missing.Kind)
	require.Equal(t, int64(10), missing.ExpectedQty)
	require.Equal(t, int64(8), missing.ReceivedQty) // 7 received + 1 lost: the lost unit did arrive
	require.Equal(t, int64(2), missing.Quantity)
	require.Equal(t, "SUP-line-1", missing.SupplierSKU)
	require.Equal(t, "Supplier text line-1", missing.SupplierDescription)
	require.Equal(t, completedAt, missing.OccurredAt)
	require.Equal(t, procurement.SupplierIssueStatusOpen, missing.Status)

	damagedScan := got[1]
	require.Equal(t, procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceBookingScan, SourceID: "scan-3"}, damagedScan.Ref)
	require.Equal(t, procurement.SupplierIssueDamaged, damagedScan.Kind)
	require.Equal(t, int64(2), damagedScan.Quantity)
	require.Equal(t, "crushed box", damagedScan.Reason)
	require.Equal(t, "ana", damagedScan.ReportedBy)
	require.Equal(t, "line-2", *damagedScan.LineItemID)
	require.Equal(t, "Supplier text line-2", damagedScan.SupplierDescription)
	require.Equal(t, procurement.SupplierIssueStatusClaimed, damagedScan.Status)
	require.Equal(t, "emailed", damagedScan.StatusNote)
	require.Equal(t, &updatedAt, damagedScan.StatusUpdatedAt)

	require.Equal(t, procurement.SupplierIssueLost, got[2].Kind)
	require.Equal(t, "scan-4", got[2].Ref.SourceID)

	flagged := got[3]
	require.Equal(t, procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceReceivingDiscrepancy, SourceID: "disc-1"}, flagged.Ref)
	require.Equal(t, procurement.SupplierIssueDamaged, flagged.Kind)
	require.Equal(t, int64(1), flagged.Quantity)
	require.Equal(t, "leaking", flagged.Reason)
	require.Equal(t, "sku-2", *flagged.SKUID)
	require.Equal(t, flaggedAt, flagged.OccurredAt)
}

func TestCollectSupplierIssues_noCompletedBookingMeansNothingIsMissingYet(t *testing.T) {
	in := procurement.SupplierIssueInputs{LineItems: []procurement.PurchaseOrderLineItem{productLine("line-1", "sku-1", 10)}}

	require.Empty(t, procurement.CollectSupplierIssues(in))
}

func TestSupplierIssueStatus_Valid(t *testing.T) {
	for _, s := range []procurement.SupplierIssueStatus{procurement.SupplierIssueStatusOpen, procurement.SupplierIssueStatusClaimed, procurement.SupplierIssueStatusResolved} {
		require.True(t, s.Valid(), s)
	}
	require.False(t, procurement.SupplierIssueStatus("lost").Valid())
}

func TestSupplierIssueSource_Valid(t *testing.T) {
	for _, s := range []procurement.SupplierIssueSource{procurement.SupplierIssueSourceBookingShortfall, procurement.SupplierIssueSourceBookingScan, procurement.SupplierIssueSourceReceivingDiscrepancy} {
		require.True(t, s.Valid(), s)
	}
	require.False(t, procurement.SupplierIssueSource("invoice").Valid())
}

func TestSupplierIssueRef_Find(t *testing.T) {
	issues := []procurement.SupplierIssue{{Ref: procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceBookingScan, SourceID: "scan-1"}}}

	found, ok := procurement.FindSupplierIssue(issues, procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceBookingScan, SourceID: "scan-1"})
	require.True(t, ok)
	require.Equal(t, "scan-1", found.Ref.SourceID)

	_, ok = procurement.FindSupplierIssue(issues, procurement.SupplierIssueRef{Source: procurement.SupplierIssueSourceBookingShortfall, SourceID: "scan-1"})
	require.False(t, ok)
}
