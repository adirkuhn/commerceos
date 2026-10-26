package oms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSalesOrder_Editable(t *testing.T) {
	manual := func(status SalesOrderStatus) SalesOrder {
		return SalesOrder{Source: SalesOrderSourceManual, Status: status}
	}
	for _, s := range []SalesOrderStatus{SalesOrderStatusPending, SalesOrderStatusProcessing, SalesOrderStatusOnHold} {
		assert.NoError(t, manual(s).Editable(), s)
	}
	for _, s := range []SalesOrderStatus{SalesOrderStatusShipped, SalesOrderStatusCompleted, SalesOrderStatusCancelled, SalesOrderStatusRefunded, SalesOrderStatusFailed} {
		assert.ErrorIs(t, manual(s).Editable(), ErrSalesOrderNotEditable, s)
	}

	fulfilled := manual(SalesOrderStatusProcessing)
	fulfilled.Fulfilled = true
	assert.ErrorIs(t, fulfilled.Editable(), ErrSalesOrderNotEditable)

	woo := SalesOrder{Source: SalesOrderSourceWooCommerce, Status: SalesOrderStatusProcessing}
	assert.ErrorIs(t, woo.Editable(), ErrSalesOrderNotEditable)
}
