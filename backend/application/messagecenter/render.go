package messagecenter

import (
	"html"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

// templateData is what a template renders against: every payload field
// under its own JSON key (e.g. {{status_code}}, or {{#items}} for a list),
// generic across trigger subjects, plus the audience's own values (e.g.
// {{customer_name}}), which never live in an event payload.
func templateData(payload, audienceVars map[string]any) map[string]any {
	data := make(map[string]any, len(payload)+len(audienceVars))
	for k, v := range payload {
		data[k] = v
	}
	for k, v := range audienceVars {
		data[k] = v
	}
	return data
}

// renderBody renders a template body for ch: HTML-escaped values for email
// (its body is HTML, so a product name with "&" or "<" must not break it),
// raw values for plain-text channels.
func renderBody(body string, data map[string]any, ch messaging.Channel) string {
	if ch == messaging.ChannelEmail {
		return mcDomain.Render(body, data, html.EscapeString)
	}
	return mcDomain.Render(body, data, plainText)
}

// renderSubject renders an email subject -- plain text, never escaped.
func renderSubject(subject string, data map[string]any) string {
	return mcDomain.Render(subject, data, plainText)
}

func plainText(s string) string { return s }
