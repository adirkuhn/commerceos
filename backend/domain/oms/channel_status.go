package oms

// AcceptsChannelStatus decides whether a status read back from a sales
// channel (e.g. a WooCommerce order import) may replace this local status.
//
// The channel is not always ahead of us: during an outage our status pushes
// fail, and Shipped is pushed to Woo as "on-hold" (Woo has no shipped
// status), so a re-import would read either a stale pre-shipment status or
// the echo of our own push. Woo also has no partially-refunded status, so
// such an order stays "processing" there. Once an order has moved past the
// pre-shipment stage locally (Shipped, Completed, PartiallyRefunded) it
// never moves back to pending/processing/on-hold from a channel. A
// terminal outcome decided on the channel (cancelled, refunded, failed)
// always wins.
func (current SalesOrderStatus) AcceptsChannelStatus(incoming SalesOrderStatus) bool {
	if incoming == current {
		return false
	}
	if incoming.isChannelTerminalOutcome() {
		return true
	}
	return !(current.isBeyondChannelPreShipment() && incoming.isPreShipment())
}

// isBeyondChannelPreShipment lists the local statuses a channel can't
// express or lags behind on.
func (s SalesOrderStatus) isBeyondChannelPreShipment() bool {
	switch s {
	case SalesOrderStatusShipped, SalesOrderStatusCompleted, SalesOrderStatusPartiallyRefunded:
		return true
	default:
		return false
	}
}

func (s SalesOrderStatus) isPreShipment() bool {
	switch s {
	case SalesOrderStatusPending, SalesOrderStatusProcessing, SalesOrderStatusOnHold:
		return true
	default:
		return false
	}
}

func (s SalesOrderStatus) isChannelTerminalOutcome() bool {
	switch s {
	case SalesOrderStatusCancelled, SalesOrderStatusRefunded, SalesOrderStatusFailed:
		return true
	default:
		return false
	}
}
