package messagecenter

import (
	"fmt"
	"strings"
)

// RenderTemplate substitutes {{token}} placeholders in body using vars.
// A placeholder with no matching key is left as-is (unresolved -- rather
// than silently blanked) so a merchant's typo in a template is visible in
// the sent message instead of vanishing without a trace.
func RenderTemplate(body string, vars map[string]string) string {
	replacements := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		replacements = append(replacements, "{{"+k+"}}", v)
	}
	return strings.NewReplacer(replacements...).Replace(body)
}

// buildTemplateVars turns a triggering event's payload plus the resolved
// recipient into the variable map RenderTemplate substitutes. Every payload
// field is exposed directly under its own JSON key (e.g. {{status_code}},
// {{sales_order_id}}) -- generic across every trigger subject, no
// per-event-type mapping to maintain -- plus {{customer_name}} for the one
// field that never lives in an event payload itself.
func buildTemplateVars(payload map[string]any, customerName string) map[string]string {
	vars := map[string]string{"customer_name": customerName}
	for k, v := range payload {
		vars[k] = fmt.Sprint(v)
	}
	return vars
}
