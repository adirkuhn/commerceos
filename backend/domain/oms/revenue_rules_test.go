package oms_test

import (
	"testing"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
)

func TestSalesOrderStatus_CountsAsRevenue(t *testing.T) {
	for _, s := range []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusOnHold, omsDomain.SalesOrderStatusProcessing,
		omsDomain.SalesOrderStatusShipped, omsDomain.SalesOrderStatusCompleted,
	} {
		assert.True(t, s.CountsAsRevenue(), s)
	}
	for _, s := range []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusCancelled,
		omsDomain.SalesOrderStatusRefunded, omsDomain.SalesOrderStatusFailed,
	} {
		assert.False(t, s.CountsAsRevenue(), s)
	}
}

// RevenueCountingStatuses is the set a repository's SQL filter uses -- it
// must be exactly the statuses CountsAsRevenue accepts, so the rule lives
// in one place (a hand-maintained copy is how shipped orders silently fell
// out of every revenue report).
func TestRevenueCountingStatuses_MatchesCountsAsRevenue(t *testing.T) {
	var want []omsDomain.SalesOrderStatus
	for _, s := range omsDomain.SalesOrderStatuses() {
		if s.CountsAsRevenue() {
			want = append(want, s)
		}
	}
	assert.ElementsMatch(t, want, omsDomain.RevenueCountingStatuses())
	assert.Contains(t, omsDomain.RevenueCountingStatuses(), omsDomain.SalesOrderStatusShipped)
}

// SalesOrderStatuses must list every status the sales_orders CHECK
// constraint allows (see migration 142) -- a status missing here would be
// invisible to every set derived from it.
func TestSalesOrderStatuses_ListsEveryStatus(t *testing.T) {
	assert.ElementsMatch(t, []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold,
		omsDomain.SalesOrderStatusShipped, omsDomain.SalesOrderStatusCompleted, omsDomain.SalesOrderStatusCancelled,
		omsDomain.SalesOrderStatusRefunded, omsDomain.SalesOrderStatusFailed, omsDomain.SalesOrderStatusPartiallyRefunded,
	}, omsDomain.SalesOrderStatuses())
}
