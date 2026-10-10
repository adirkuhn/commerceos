package messagecenter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestOperator_Valid(t *testing.T) {
	valid := []messagecenter.Operator{
		messagecenter.OperatorEq, messagecenter.OperatorNeq, messagecenter.OperatorIn,
		messagecenter.OperatorGt, messagecenter.OperatorLt,
	}
	for _, o := range valid {
		assert.True(t, o.Valid(), "expected %q to be valid", o)
	}
	assert.False(t, messagecenter.Operator("bogus").Valid())
}

func TestValidTriggerSubject(t *testing.T) {
	assert.True(t, messagecenter.ValidTriggerSubject("shipping.shipment.fulfillment_signal"))
	assert.False(t, messagecenter.ValidTriggerSubject("bogus.subject.here"))
}

func TestAutomationRule_Matches_noConditionsAlwaysMatches(t *testing.T) {
	r := messagecenter.AutomationRule{}
	assert.True(t, r.Matches(map[string]any{"anything": 1}))
}

func TestAutomationRule_Matches_eq(t *testing.T) {
	r := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "status_main", Operator: messagecenter.OperatorEq, Value: "delivered"},
	}}
	assert.True(t, r.Matches(map[string]any{"status_main": "delivered"}))
	assert.False(t, r.Matches(map[string]any{"status_main": "in_transit"}))
}

func TestAutomationRule_Matches_neq(t *testing.T) {
	r := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "status_main", Operator: messagecenter.OperatorNeq, Value: "delivered"},
	}}
	assert.False(t, r.Matches(map[string]any{"status_main": "delivered"}))
	assert.True(t, r.Matches(map[string]any{"status_main": "in_transit"}))
}

// TestAutomationRule_Matches_in_shipmentDeliveredToCollectPoint mirrors the
// user's own example: MyParcel status codes 8/19 mean "left at a collect
// point" rather than "handed to the recipient" -- see
// application/whatsapp/notify_shipment_delivery.go's collectPointStatusCodes.
func TestAutomationRule_Matches_in_shipmentDeliveredToCollectPoint(t *testing.T) {
	r := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "status_code", Operator: messagecenter.OperatorIn, Value: []any{float64(8), float64(19)}},
	}}
	assert.True(t, r.Matches(map[string]any{"status_code": float64(8)}))
	assert.True(t, r.Matches(map[string]any{"status_code": float64(19)}))
	assert.False(t, r.Matches(map[string]any{"status_code": float64(7)}))
}

func TestAutomationRule_Matches_gtLt(t *testing.T) {
	r := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "quantity", Operator: messagecenter.OperatorGt, Value: float64(5)},
	}}
	assert.True(t, r.Matches(map[string]any{"quantity": float64(10)}))
	assert.False(t, r.Matches(map[string]any{"quantity": float64(3)}))

	rLt := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "quantity", Operator: messagecenter.OperatorLt, Value: float64(5)},
	}}
	assert.True(t, rLt.Matches(map[string]any{"quantity": float64(3)}))
	assert.False(t, rLt.Matches(map[string]any{"quantity": float64(10)}))
}

func TestAutomationRule_Matches_missingFieldNeverMatches(t *testing.T) {
	r := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "status_code", Operator: messagecenter.OperatorEq, Value: float64(8)},
	}}
	assert.False(t, r.Matches(map[string]any{"other_field": 1}))
}

func TestAutomationRule_Matches_multipleConditionsAreAND(t *testing.T) {
	r := messagecenter.AutomationRule{Conditions: []messagecenter.Condition{
		{Field: "status_main", Operator: messagecenter.OperatorEq, Value: "delivered"},
		{Field: "kind", Operator: messagecenter.OperatorEq, Value: "shipment_status"},
	}}
	assert.True(t, r.Matches(map[string]any{"status_main": "delivered", "kind": "shipment_status"}))
	assert.False(t, r.Matches(map[string]any{"status_main": "delivered", "kind": "order_processed"}))
}

func TestNormalizeExtraRecipients(t *testing.T) {
	got, err := messagecenter.NormalizeExtraRecipients([]string{" ops@example.nl ", "", "OPS@example.nl", "boss@example.nl"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ops@example.nl", "boss@example.nl"}, got)

	got, err = messagecenter.NormalizeExtraRecipients(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{}, got)
}

func TestNormalizeExtraRecipients_rejectsInvalidAddress(t *testing.T) {
	_, err := messagecenter.NormalizeExtraRecipients([]string{"not-an-email"})
	assert.ErrorIs(t, err, messagecenter.ErrInvalidExtraRecipient)

	_, err = messagecenter.NormalizeExtraRecipients([]string{"Jane <jane@example.nl>"})
	assert.ErrorIs(t, err, messagecenter.ErrInvalidExtraRecipient, "bare addresses only")
}

func TestNormalizeExtraRecipients_rejectsTooMany(t *testing.T) {
	many := make([]string, 21)
	for i := range many {
		many[i] = fmt.Sprintf("user%d@example.nl", i)
	}
	_, err := messagecenter.NormalizeExtraRecipients(many)
	assert.ErrorIs(t, err, messagecenter.ErrTooManyExtraRecipients)
}

func TestAudienceFor(t *testing.T) {
	assert.Equal(t, messagecenter.Audience{Kind: messagecenter.AudienceCustomer}, messagecenter.AudienceFor("oms.sales_order.status_changed"))
	assert.Equal(t, messagecenter.Audience{Kind: messagecenter.AudienceStaff, StaffCategory: "stock"},
		messagecenter.AudienceFor(messagecenter.TriggerStockCloseToExpiry))
	assert.True(t, messagecenter.ValidTriggerSubject(messagecenter.TriggerStockCloseToExpiry))
}

func TestTriggerStockCloseToExpiry_matchesTheEventSubject(t *testing.T) {
	assert.Equal(t, events.StockCloseToExpiryReported{}.EventName(), messagecenter.TriggerStockCloseToExpiry)
}
