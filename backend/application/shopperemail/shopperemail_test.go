package shopperemail_test

import (
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/shopperemail"
	"github.com/adirkuhn/commerceos/domain/locale"
)

func TestEveryLanguageHasEveryTextWithTheSamePlaceholders(t *testing.T) {
	en := shopperemail.Texts(locale.English)
	placeholders := func(s string) []string {
		var out []string
		for _, m := range regexp.MustCompile(`\{\{(\w+)\}\}`).FindAllStringSubmatch(s, -1) {
			out = append(out, m[1])
		}
		sort.Strings(out)
		return out
	}
	for _, l := range []locale.Locale{locale.Portuguese, locale.Dutch} {
		texts := shopperemail.Texts(l)
		require.Len(t, texts, len(en), l)
		for key, text := range en {
			assert.NotEmpty(t, texts[key], "%s %s", l, key)
			assert.Equal(t, placeholders(text), placeholders(texts[key]), "%s %s", l, key)
		}
	}
}

func TestSignInEmail(t *testing.T) {
	subject, body := shopperemail.SignIn(locale.Portuguese, "https://shop.example/login?token=a&b")
	assert.Equal(t, "Seu link de acesso", subject)
	assert.Contains(t, body, `<a href="https://shop.example/login?token=a&amp;b">Entrar</a>`)
	assert.Contains(t, body, "expira em 15 minutos")
}

func TestNewsletterConfirmationEmail(t *testing.T) {
	subject, body := shopperemail.NewsletterConfirmation(locale.Dutch, "Café <&>", "https://c", "https://u")
	assert.Equal(t, "Bevestig je inschrijving bij Café <&>", subject, "subjects are plain text")
	assert.Contains(t, body, "e-mails van Café &lt;&amp;&gt; wilt")
	assert.Contains(t, body, `<a href="https://c">Ja, schrijf me in</a>`)
	assert.Contains(t, body, `<a href="https://u">Uitschrijven</a>`)
}

func TestUnknownLanguageFallsBackToEnglish(t *testing.T) {
	subject, _ := shopperemail.SignIn("de", "https://x")
	assert.Equal(t, "Your sign-in link", subject)
}
