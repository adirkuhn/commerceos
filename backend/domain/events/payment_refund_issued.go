package events

// PaymentRefundIssued is raised whenever CompletePickingUseCase's replay
// successfully refunds a PickScanActionRefund row through the order's
// payment provider (Stripe/Revolut) -- see application/oms.
// CompletePickingUseCase's doc. Carries a value snapshot (unlike
// SalesOrderStatusChanged's id-only shape) since a refund's amount/quantity
// at the moment it happened is itself the fact worth recording, not
// something a consumer should re-derive from current state.
type PaymentRefundIssued struct {
	Base
	TenantID         string `json:"tenant_id"`
	SalesOrderID     string `json:"sales_order_id"`
	LineItemID       string `json:"line_item_id"`
	SKUID            string `json:"sku_id"`
	Quantity         int64  `json:"quantity"`
	AmountMinorUnits int64  `json:"amount_minor_units"`
}

func (PaymentRefundIssued) EventName() string { return "payments.refund.issued" }
