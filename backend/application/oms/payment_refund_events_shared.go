package oms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// enqueuePaymentRefundIssued writes a payments.refund.issued outbox row in
// the caller's current transaction -- see sales_order_status_changed_shared.
// go's enqueueSalesOrderStatusChanged for the identical pattern/rationale
// (must be called from inside the same tenant.WithTenant block as the write
// it documents, so event and write commit atomically).
func enqueuePaymentRefundIssued(ctx context.Context, outboxRepo outbox.Repository, salesOrderID, lineItemID, skuID string, quantity, amountMinorUnits int64) error {
	event := events.PaymentRefundIssued{
		Base: events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()}, TenantID: tenant.IDFromContext(ctx),
		SalesOrderID: salesOrderID, LineItemID: lineItemID, SKUID: skuID, Quantity: quantity, AmountMinorUnits: amountMinorUnits,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}

// enqueuePaymentRefundManualReviewRequired is enqueuePaymentRefundIssued's
// manual-review-path sibling -- same pattern, different event.
func enqueuePaymentRefundManualReviewRequired(ctx context.Context, outboxRepo outbox.Repository, salesOrderID, lineItemID, skuID string, quantity, amountMinorUnits int64) error {
	event := events.PaymentRefundManualReviewRequired{
		Base: events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()}, TenantID: tenant.IDFromContext(ctx),
		SalesOrderID: salesOrderID, LineItemID: lineItemID, SKUID: skuID, Quantity: quantity, AmountMinorUnits: amountMinorUnits,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
