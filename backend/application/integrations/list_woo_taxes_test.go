package integrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestDedupeWooTaxClasses_collapsesRepeatedClassesAcrossCountries(t *testing.T) {
	rates := []integrationsDomain.WooTaxRate{
		{ID: "1", Country: "NL", Rate: "21.0000", Class: "standard"},
		{ID: "2", Country: "BE", Rate: "21.0000", Class: "standard"},
		{ID: "3", Country: "NL", Rate: "9.0000", Class: "reduced-rate"},
	}

	got := appintegrations.DedupeWooTaxClasses(rates)

	assert.Len(t, got, 2)
}

func TestDedupeWooTaxClasses_sortsStandardFirstThenAlphabeticalByLabel(t *testing.T) {
	rates := []integrationsDomain.WooTaxRate{
		{ID: "1", Country: "NL", Rate: "0.0000", Class: "zero-rate"},
		{ID: "2", Country: "NL", Rate: "9.0000", Class: "reduced-rate"},
		{ID: "3", Country: "NL", Rate: "21.0000", Class: "standard"},
	}

	got := appintegrations.DedupeWooTaxClasses(rates)

	assert.Equal(t, []appintegrations.WooTaxClassOption{
		{Slug: "standard", Label: "Standard"},
		{Slug: "reduced-rate", Label: "Reduced Rate"},
		{Slug: "zero-rate", Label: "Zero Rate"},
	}, got)
}

func TestDedupeWooTaxClasses_emptySlugAlsoLabeledStandard(t *testing.T) {
	rates := []integrationsDomain.WooTaxRate{
		{ID: "1", Country: "NL", Rate: "21.0000", Class: ""},
	}

	got := appintegrations.DedupeWooTaxClasses(rates)

	assert.Equal(t, []appintegrations.WooTaxClassOption{{Slug: "", Label: "Standard"}}, got)
}

func TestDedupeWooTaxClasses_emptyInputReturnsEmpty(t *testing.T) {
	got := appintegrations.DedupeWooTaxClasses(nil)
	assert.Empty(t, got)
}
