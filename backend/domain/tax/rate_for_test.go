package tax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateFor(t *testing.T) {
	classes := []TaxClass{{ID: "standard", IsDefault: true}, {ID: "reduced"}}
	rates := []TaxRate{
		{TaxClassID: "standard", Country: "NL", RateBasisPoints: 2100},
		{TaxClassID: "reduced", Country: "NL", RateBasisPoints: 900},
	}
	reduced := "reduced"

	bp, ok := RateFor(classes, rates, &reduced, "NL")
	assert.True(t, ok)
	assert.Equal(t, int32(900), bp)

	bp, ok = RateFor(classes, rates, nil, "nl")
	assert.True(t, ok, "no class on the SKU uses the default class; country is case-insensitive")
	assert.Equal(t, int32(2100), bp)

	_, ok = RateFor(classes, rates, &reduced, "BE")
	assert.False(t, ok, "no rate for that country")

	_, ok = RateFor([]TaxClass{{ID: "reduced"}}, rates, nil, "NL")
	assert.False(t, ok, "no class and no default class")
}
