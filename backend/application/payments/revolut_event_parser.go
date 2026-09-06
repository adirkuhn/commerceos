package payments

import (
	"encoding/json"
	"fmt"
)

// revolutOrderCompletedEvent is the only Revolut webhook payload shape this
// codebase handles. UNCONFIRMED against a real Revolut account -- field
// names per published docs only, same caveat as RevolutVerifier and
// infrastructure/revolut.Client.
type revolutOrderCompletedEvent struct {
	Event               string `json:"event"`
	OrderID             string `json:"order_id"`
	MerchantOrderExtRef string `json:"merchant_order_ext_ref"`
}

const revolutEventOrderCompleted = "ORDER_COMPLETED"

// RevolutEventParser implements PaymentEventParser for Revolut.
type RevolutEventParser struct{}

func NewRevolutEventParser() *RevolutEventParser { return &RevolutEventParser{} }

func (RevolutEventParser) Parse(rawPayload []byte) (PaymentEvent, error) {
	var event revolutOrderCompletedEvent
	if err := json.Unmarshal(rawPayload, &event); err != nil {
		return PaymentEvent{}, fmt.Errorf("revolutEventParser: decode event: %w", err)
	}
	if event.Event != revolutEventOrderCompleted {
		return PaymentEvent{Handled: false}, nil
	}
	return PaymentEvent{Handled: true, OrderNumber: event.MerchantOrderExtRef, ExternalPaymentID: event.OrderID}, nil
}
