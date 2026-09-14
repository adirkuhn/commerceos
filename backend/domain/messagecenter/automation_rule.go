package messagecenter

import (
	"fmt"
	"strconv"
	"time"

	"github.com/adirkuhn/commerceos/domain/messaging"
)

type Operator string

const (
	OperatorEq  Operator = "eq"
	OperatorNeq Operator = "neq"
	OperatorIn  Operator = "in"
	OperatorGt  Operator = "gt"
	OperatorLt  Operator = "lt"
)

func (o Operator) Valid() bool {
	switch o {
	case OperatorEq, OperatorNeq, OperatorIn, OperatorGt, OperatorLt:
		return true
	default:
		return false
	}
}

// Condition is one "event payload field compares to value" check.
// AutomationRule.Conditions is a flat AND list -- v1 has no OR/grouping, see
// this package's own doc comment on why that's deliberate.
type Condition struct {
	Field    string   `json:"field"`
	Operator Operator `json:"operator"`
	Value    any      `json:"value"`
}

// KnownTriggerSubjects is every domain event subject an AutomationRule can
// fire on -- see domain/events for the structs these come from. No single
// source of truth exists there (each event's own EventName() is the only
// place its subject lives), so this list is a deliberate, separately
// maintained mirror -- add a line here whenever a new domain event should
// be automatable.
var KnownTriggerSubjects = []string{
	"pms.product.deleted",
	"storefront.order.placed",
	"pms.category.changed",
	"pms.product.changed",
	"payments.refund.manual_review_required",
	"payments.refund.issued",
	"integrations.connection.connected",
	"oms.sales_order.status_changed",
	"inventory.stock.changed",
	"shipping.shipment.fulfillment_signal",
}

func ValidTriggerSubject(subject string) bool {
	for _, s := range KnownTriggerSubjects {
		if s == subject {
			return true
		}
	}
	return false
}

// AutomationRule is "when TriggerSubject fires and every Condition matches
// the event's payload, send TemplateID" -- the engine that evaluates these
// (application/messagecenter.EvaluateAutomationRulesUseCase) is a later
// slice; this is just the configuration shape + the pure matching logic.
type AutomationRule struct {
	ID             string
	TenantID       string
	Name           string
	TriggerSubject string
	Conditions     []Condition
	TemplateID     string
	// ChannelOverride, nil meaning "use the template's own Channels",
	// lets one rule narrow a multi-channel template down to a single
	// channel for this specific trigger -- e.g. a template enabled for
	// email+whatsapp, but this rule only ever sends it by whatsapp.
	ChannelOverride *messaging.Channel
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Matches reports whether every condition holds against payload (flat AND).
// An empty Conditions list always matches -- "fire on every occurrence of
// this event" is a valid rule needing no conditions.
func (r AutomationRule) Matches(payload map[string]any) bool {
	for _, c := range r.Conditions {
		if !c.matches(payload) {
			return false
		}
	}
	return true
}

func (c Condition) matches(payload map[string]any) bool {
	actual, ok := payload[c.Field]
	if !ok {
		return false
	}
	switch c.Operator {
	case OperatorEq:
		return equalValue(actual, c.Value)
	case OperatorNeq:
		return !equalValue(actual, c.Value)
	case OperatorIn:
		values, ok := c.Value.([]any)
		if !ok {
			return false
		}
		for _, v := range values {
			if equalValue(actual, v) {
				return true
			}
		}
		return false
	case OperatorGt, OperatorLt:
		af, aok := toFloat(actual)
		vf, vok := toFloat(c.Value)
		if !aok || !vok {
			return false
		}
		if c.Operator == OperatorGt {
			return af > vf
		}
		return af < vf
	default:
		return false
	}
}

// equalValue compares via string representation -- sidesteps float64 (every
// number decoded from JSON) vs int (a number set directly in a Go literal,
// e.g. in a test) never comparing equal through ==, without hand-rolling a
// type switch for every JSON-representable type.
func equalValue(a, b any) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
