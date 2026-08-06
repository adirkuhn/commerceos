package procurement_test

import (
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/stretchr/testify/assert"
)

func TestDetectCurrencyCode_findsStandaloneCode(t *testing.T) {
	code, ok := appprocurement.DetectCurrencyCode("Moeda\nEUR\n1,000000")
	assert.True(t, ok)
	assert.Equal(t, "EUR", code)
}

func TestDetectCurrencyCode_ignoresCodeInsideAnotherWord(t *testing.T) {
	// "EURO" contains "EUR" but isn't the standalone code -- must not match.
	code, ok := appprocurement.DetectCurrencyCode("Price in EUROS only, see EUROPE section")
	assert.False(t, ok)
	assert.Empty(t, code)
}

func TestDetectCurrencyCode_returnsFalseWhenNoneFound(t *testing.T) {
	_, ok := appprocurement.DetectCurrencyCode("no currency mentioned here")
	assert.False(t, ok)
}

func TestDetectCurrencyCode_findsFirstMatchAmongMultiple(t *testing.T) {
	code, ok := appprocurement.DetectCurrencyCode("quoted in USD, converted from GBP")
	assert.True(t, ok)
	assert.Equal(t, "USD", code)
}

func TestDetectCurrencyCode_fallsBackToEuroSymbolWhenNoISOCodePrinted(t *testing.T) {
	// Real-world case: a Dutch invoice that prints amounts like "323.64 €"
	// and never spells out "EUR" anywhere on the page.
	code, ok := appprocurement.DetectCurrencyCode("Untaxed Amount 323.64 €\nVAT 9% 29.13 €\nTotal 352.77 €")
	assert.True(t, ok)
	assert.Equal(t, "EUR", code)
}

func TestDetectCurrencyCode_prefersISOCodeOverSymbolWhenBothPresent(t *testing.T) {
	code, ok := appprocurement.DetectCurrencyCode("Totaalbedrag EUR 405,04")
	assert.True(t, ok)
	assert.Equal(t, "EUR", code)
}

func TestDetectCurrencyCode_poundSymbolMapsToGBP(t *testing.T) {
	code, ok := appprocurement.DetectCurrencyCode("Total due: £120.00")
	assert.True(t, ok)
	assert.Equal(t, "GBP", code)
}
