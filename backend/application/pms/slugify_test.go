package pms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/application/pms"
)

func TestSlugify_lowercasesAndDashesSpaces(t *testing.T) {
	assert.Equal(t, "wireless-headphones-pro", pms.Slugify("Wireless Headphones Pro"))
}

func TestSlugify_collapsesPunctuationToSingleDash(t *testing.T) {
	assert.Equal(t, "usb-c-hub-7-in-1", pms.Slugify("USB-C Hub 7-in-1!!"))
}

func TestSlugify_trimsLeadingAndTrailingDashes(t *testing.T) {
	assert.Equal(t, "product", pms.Slugify("  Product!!  "))
}

func TestSlugify_emptyInputYieldsEmptyString(t *testing.T) {
	assert.Equal(t, "", pms.Slugify("   ---   "))
}
