package oms_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestSalesOrder_PaymentStatus(t *testing.T) {
	paidAt := time.Date(2026, 10, 2, 14, 31, 0, 0, time.UTC)
	cases := []struct {
		name     string
		status   omsDomain.SalesOrderStatus
		datePaid *time.Time
		want     omsDomain.PaymentStatus
	}{
		{"pending: no payment yet", omsDomain.SalesOrderStatusPending, nil, omsDomain.PaymentStatusAwaitingPayment},
		{"pending with a paid date (payment landed, status not synced yet)", omsDomain.SalesOrderStatusPending, &paidAt, omsDomain.PaymentStatusPaid},
		{"on-hold: offline payment awaiting confirmation", omsDomain.SalesOrderStatusOnHold, nil, omsDomain.PaymentStatusAwaitingConfirmation},
		{"on-hold already confirmed paid", omsDomain.SalesOrderStatusOnHold, &paidAt, omsDomain.PaymentStatusPaid},
		{"processing means paid", omsDomain.SalesOrderStatusProcessing, nil, omsDomain.PaymentStatusPaid},
		{"shipped", omsDomain.SalesOrderStatusShipped, nil, omsDomain.PaymentStatusPaid},
		{"completed", omsDomain.SalesOrderStatusCompleted, nil, omsDomain.PaymentStatusPaid},
		{"failed", omsDomain.SalesOrderStatusFailed, nil, omsDomain.PaymentStatusFailed},
		{"failed even with a stale paid date", omsDomain.SalesOrderStatusFailed, &paidAt, omsDomain.PaymentStatusFailed},
		{"refunded", omsDomain.SalesOrderStatusRefunded, &paidAt, omsDomain.PaymentStatusRefunded},
		{"partially refunded", omsDomain.SalesOrderStatusPartiallyRefunded, &paidAt, omsDomain.PaymentStatusPartiallyRefunded},
		{"cancelled before paying", omsDomain.SalesOrderStatusCancelled, nil, omsDomain.PaymentStatusCancelledUnpaid},
		{"cancelled after paying (not refunded yet)", omsDomain.SalesOrderStatusCancelled, &paidAt, omsDomain.PaymentStatusPaid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := omsDomain.SalesOrder{Status: tc.status, DatePaid: tc.datePaid}
			assert.Equal(t, tc.want, o.PaymentStatus())
		})
	}
}

func TestSalesOrder_PaymentStatus_coversEveryOrderStatus(t *testing.T) {
	for _, s := range omsDomain.SalesOrderStatuses() {
		got := omsDomain.SalesOrder{Status: s}.PaymentStatus()
		assert.Contains(t, omsDomain.PaymentStatuses(), got, s)
	}
}

func TestSalesOrder_PaymentReference(t *testing.T) {
	assert.Equal(t, "ch_123", omsDomain.SalesOrder{TransactionID: "ch_123", PaymentExternalID: "pi_456"}.PaymentReference(),
		"the channel's own transaction id wins")
	assert.Equal(t, "pi_456", omsDomain.SalesOrder{PaymentExternalID: "pi_456"}.PaymentReference(),
		"storefront card orders only carry the provider's id")
	assert.Equal(t, "", omsDomain.SalesOrder{}.PaymentReference())
}

func TestSalesOrderStatus_ExpectsPaymentRecord(t *testing.T) {
	for _, s := range []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold, omsDomain.SalesOrderStatusShipped,
		omsDomain.SalesOrderStatusCompleted, omsDomain.SalesOrderStatusRefunded, omsDomain.SalesOrderStatusPartiallyRefunded,
	} {
		assert.True(t, s.ExpectsPaymentRecord(), s)
	}
	for _, s := range []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusFailed, omsDomain.SalesOrderStatusCancelled,
	} {
		assert.False(t, s.ExpectsPaymentRecord(), s)
	}
	for _, s := range omsDomain.PaymentRecordExpectedStatuses() {
		assert.True(t, s.ExpectsPaymentRecord(), s)
	}
	assert.Len(t, omsDomain.PaymentRecordExpectedStatuses(), 6)
}
