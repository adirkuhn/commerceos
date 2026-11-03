package oms

import "testing"

func TestAcceptsChannelStatus(t *testing.T) {
	cases := []struct {
		name     string
		current  SalesOrderStatus
		incoming SalesOrderStatus
		want     bool
	}{
		{"same status is no change", SalesOrderStatusProcessing, SalesOrderStatusProcessing, false},
		{"forward move is accepted", SalesOrderStatusPending, SalesOrderStatusProcessing, true},
		{"processing to completed is accepted", SalesOrderStatusProcessing, SalesOrderStatusCompleted, true},
		{"processing to on-hold is accepted", SalesOrderStatusProcessing, SalesOrderStatusOnHold, true},

		// Shipped/Completed never regress to a pre-shipment status: the
		// channel is behind (missed push, outage) or echoing our own push
		// (Shipped goes out to Woo as on-hold).
		{"shipped ignores stale processing", SalesOrderStatusShipped, SalesOrderStatusProcessing, false},
		{"shipped ignores pending", SalesOrderStatusShipped, SalesOrderStatusPending, false},
		{"shipped ignores on-hold echo", SalesOrderStatusShipped, SalesOrderStatusOnHold, false},
		{"completed ignores stale processing", SalesOrderStatusCompleted, SalesOrderStatusProcessing, false},
		{"completed ignores on-hold", SalesOrderStatusCompleted, SalesOrderStatusOnHold, false},
		{"completed ignores pending", SalesOrderStatusCompleted, SalesOrderStatusPending, false},

		// Woo has no partially-refunded status: the order stays "processing"
		// there, so reading that back must not undo our partial refund.
		{"partially refunded ignores processing", SalesOrderStatusPartiallyRefunded, SalesOrderStatusProcessing, false},
		{"partially refunded ignores on-hold", SalesOrderStatusPartiallyRefunded, SalesOrderStatusOnHold, false},
		{"partially refunded accepts completed", SalesOrderStatusPartiallyRefunded, SalesOrderStatusCompleted, true},
		{"partially refunded accepts full refund", SalesOrderStatusPartiallyRefunded, SalesOrderStatusRefunded, true},

		{"shipped moves on to completed", SalesOrderStatusShipped, SalesOrderStatusCompleted, true},

		// Terminal outcomes decided on the channel always win.
		{"shipped accepts cancelled", SalesOrderStatusShipped, SalesOrderStatusCancelled, true},
		{"shipped accepts refunded", SalesOrderStatusShipped, SalesOrderStatusRefunded, true},
		{"shipped accepts failed", SalesOrderStatusShipped, SalesOrderStatusFailed, true},
		{"completed accepts refunded", SalesOrderStatusCompleted, SalesOrderStatusRefunded, true},
		{"completed accepts cancelled", SalesOrderStatusCompleted, SalesOrderStatusCancelled, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.current.AcceptsChannelStatus(tc.incoming); got != tc.want {
				t.Errorf("%s.AcceptsChannelStatus(%s) = %v, want %v", tc.current, tc.incoming, got, tc.want)
			}
		})
	}
}
