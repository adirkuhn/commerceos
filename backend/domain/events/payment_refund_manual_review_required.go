package events

// PaymentRefundManualReviewRequired is raised whenever CompletePickingUseCase's
// replay stages a PickScanActionRefund row's line-item bookkeeping (see
// application/oms.CompletePickingUseCase's doc) but can't call the payment
// provider's refund API for real -- either the order has no known
// PaymentProvider/PaymentExternalID (e.g. an imported order), or the API
// call itself failed. Consumers (e.g. a "needs manual refund" admin
// notification) act on SalesOrder.NeedsManualRefundReview, which this same
// replay step sets atomically with this event.
type PaymentRefundManualReviewRequired struct {
	Base
	TenantID         string `json:"tenant_id"`
	SalesOrderID     string `json:"sales_order_id"`
	LineItemID       string `json:"line_item_id"`
	SKUID            string `json:"sku_id"`
	Quantity         int64  `json:"quantity"`
	AmountMinorUnits int64  `json:"amount_minor_units"`
}

func (PaymentRefundManualReviewRequired) EventName() string {
	return "payments.refund.manual_review_required"
}
