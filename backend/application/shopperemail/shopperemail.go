// Package shopperemail writes the texts the backend sends shoppers (sign-in
// link and newsletter confirmation emails, the payment-page description)
// in their store's language. Texts live in
// locales/<lang>.json -- the same shape as the frontends' translation
// files -- so translators edit them the same way (see docs/i18n.md).
package shopperemail

import (
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/adirkuhn/commerceos/domain/locale"
)

//go:embed locales/*.json
var files embed.FS

var catalogs = func() map[locale.Locale]map[string]string {
	out := map[locale.Locale]map[string]string{}
	for _, l := range []locale.Locale{locale.English, locale.Portuguese, locale.Dutch} {
		raw, err := files.ReadFile("locales/" + string(l) + ".json")
		if err != nil {
			panic(fmt.Sprintf("shopperemail: missing %s texts: %v", l, err))
		}
		var nested map[string]map[string]string
		if err := json.Unmarshal(raw, &nested); err != nil {
			panic(fmt.Sprintf("shopperemail: bad %s texts: %v", l, err))
		}
		flat := map[string]string{}
		for group, texts := range nested {
			for key, text := range texts {
				flat[group+"."+key] = text
			}
		}
		out[l] = flat
	}
	return out
}()

// Texts returns a language's texts by "group.key" (English for unknown
// languages).
func Texts(l locale.Locale) map[string]string {
	if c, ok := catalogs[l]; ok {
		return c
	}
	return catalogs[locale.English]
}

func text(l locale.Locale, key string, params map[string]string) string {
	t := Texts(l)[key]
	if t == "" {
		t = catalogs[locale.English][key]
	}
	for name, value := range params {
		t = strings.ReplaceAll(t, "{{"+name+"}}", value)
	}
	return t
}

// escaped fills a text for HTML: the text and the values both escaped.
func escaped(l locale.Locale, key string, params map[string]string) string {
	safe := make(map[string]string, len(params))
	for k, v := range params {
		safe[k] = html.EscapeString(v)
	}
	return text(l, key, safe)
}

// SignIn is the magic-link email.
func SignIn(l locale.Locale, link string) (subject, htmlBody string) {
	return text(l, "signIn.subject", nil),
		fmt.Sprintf(`<p>%s</p><p><a href="%s">%s</a></p>`,
			escaped(l, "signIn.intro", nil), html.EscapeString(link), escaped(l, "signIn.button", nil))
}

// NewsletterConfirmation is the double opt-in email.
func NewsletterConfirmation(l locale.Locale, storeName, confirmURL, unsubscribeURL string) (subject, htmlBody string) {
	store := map[string]string{"store": storeName}
	return text(l, "newsletterConfirm.subject", store),
		fmt.Sprintf(`<p>%s</p><p><a href="%s">%s</a></p><p>%s</p><p style="font-size:12px;color:#666"><a href="%s">%s</a></p>`,
			escaped(l, "newsletterConfirm.intro", store), html.EscapeString(confirmURL), escaped(l, "newsletterConfirm.button", nil),
			escaped(l, "newsletterConfirm.ignore", nil), html.EscapeString(unsubscribeURL), escaped(l, "newsletterConfirm.unsubscribe", nil))
}

// PaymentDescription is what the shopper's payment page (Stripe, Revolut)
// shows for an order, e.g. "Bestelling #1001".
func PaymentDescription(l locale.Locale, orderNumber string) string {
	return text(l, "payment.description", map[string]string{"number": orderNumber})
}
