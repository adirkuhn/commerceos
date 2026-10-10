package messagecenter

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
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
	// ExtraRecipients are email addresses that always get this rule's
	// email, on top of whoever the trigger resolves to (the order's
	// customer, or a staff category's notification recipients). See
	// NormalizeExtraRecipients.
	ExtraRecipients []string
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const maxExtraRecipients = 20

// NormalizeExtraRecipients trims, drops blanks and case-insensitive
// duplicates, and requires bare email addresses (no "Name <addr>").
func NormalizeExtraRecipients(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, r := range raw {
		addr := strings.TrimSpace(r)
		if addr == "" || seen[strings.ToLower(addr)] {
			continue
		}
		if parsed, err := mail.ParseAddress(addr); err != nil || parsed.Address != addr {
			return nil, ErrInvalidExtraRecipient
		}
		seen[strings.ToLower(addr)] = true
		out = append(out, addr)
	}
	if len(out) > maxExtraRecipients {
		return nil, ErrTooManyExtraRecipients
	}
	return out, nil
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
