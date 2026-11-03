package events

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Naming standard (docs/events.md): {domain}.{entity}.{verb}
//   - domain: the bounded context (Domains), as in backend/domain/<domain>
//   - entity: the aggregate or record, singular snake_case (purchase_order)
//   - verb:   what happened, past tense snake_case (created, status_changed,
//     item_scanned) -- an event is a fact, never a command
//
// All lowercase; exactly three dot-separated parts.

// Domains are the subject prefixes events may use -- each one is also a
// JetStream subject (infrastructure/publishers).
var Domains = []string{
	"apikeys", "channels", "collections", "coupons", "customers", "deadletters", "fx", "iam", "integrations", "inventory",
	"messagecenter", "newsletter", "notifications", "oms", "payments", "pms", "procurement", "promotions", "quotes",
	"reviews", "scheduler", "settings", "shipping", "storefront", "suppliers", "tax", "theme", "webhook_forwarding", "whatsapp",
}

// legacySubjects predate the standard and are kept so existing consumers
// keep working. Don't add to this list.
var legacySubjects = []string{"shipping.shipment.fulfillment_signal"}

func IsLegacySubject(subject string) bool { return slices.Contains(legacySubjects, subject) }

var (
	snakePart = regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	// Irregular past tenses; regular ones end in -ed. A verb may end in a
	// particle ("rolled_back"), so any word of it counts.
	irregularPast = []string{
		"sent", "paid", "lost", "sold", "bought", "built", "held", "set", "reset", "split", "put",
		"spent", "given", "taken", "written", "chosen", "frozen", "hidden", "broken", "shown",
		"seen", "known", "won", "made", "done", "read", "run", "begun", "withdrawn",
	}
)

func ValidateSubject(subject string) error {
	parts := strings.Split(subject, ".")
	if len(parts) != 3 {
		return fmt.Errorf("event subject %q: want {domain}.{entity}.{verb}", subject)
	}
	for _, p := range parts {
		if !snakePart.MatchString(p) {
			return fmt.Errorf("event subject %q: %q is not lowercase snake_case", subject, p)
		}
	}
	if !slices.Contains(Domains, parts[0]) {
		return fmt.Errorf("event subject %q: unknown domain %q (see events.Domains)", subject, parts[0])
	}
	if !isPastTense(parts[2]) {
		return fmt.Errorf("event subject %q: verb %q is not past tense", subject, parts[2])
	}
	return nil
}

func isPastTense(verb string) bool {
	for _, w := range strings.Split(verb, "_") {
		if strings.HasSuffix(w, "ed") || slices.Contains(irregularPast, w) {
			return true
		}
	}
	return false
}
