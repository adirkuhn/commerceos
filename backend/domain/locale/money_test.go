package locale_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/locale"
)

func TestFormatMoney(t *testing.T) {
	assert.Equal(t, "€4.99", locale.English.FormatMoney(499, "EUR"))
	assert.Equal(t, "€ 4,99", locale.Dutch.FormatMoney(499, "EUR"))
	assert.Equal(t, "R$ 1.234,50", locale.Portuguese.FormatMoney(123450, "BRL"))
	assert.Equal(t, "$1,234,567.00", locale.English.FormatMoney(123456700, "USD"))
	assert.Equal(t, "£0.05", locale.English.FormatMoney(5, "GBP"))
	assert.Equal(t, "-€2.00", locale.English.FormatMoney(-200, "EUR"))
	assert.Equal(t, "CHF 3,00", locale.Dutch.FormatMoney(300, "CHF"), "unknown symbol falls back to the code")
	assert.Equal(t, "€4.99", locale.Locale("").FormatMoney(499, "EUR"), "unset means English")
}
