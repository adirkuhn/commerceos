package oms

import (
	"context"
	"fmt"
)

// OrderNumberPrefix marks numbers CommerceOS hands out itself, so they never
// read like a Woo or Bol order number.
const OrderNumberPrefix = "CO-"

// FormatOrderNumber is the human-facing number for the n-th order numbered
// here, e.g. CO-1001.
func FormatOrderNumber(n int64) string {
	return fmt.Sprintf("%s%d", OrderNumberPrefix, n)
}

// OrderNumbers hands out the next order number per tenant. Numbers are never
// reused, even after an order is deleted.
type OrderNumbers interface {
	Next(ctx context.Context) (string, error)
}
