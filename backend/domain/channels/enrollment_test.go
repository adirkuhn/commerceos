package channels_test

import (
	"testing"

	"github.com/adirkuhn/commerceos/domain/channels"
	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func TestIsProductEnrolled_allMode_enrollsEveryProduct(t *testing.T) {
	ch := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeAll}

	assert.True(t, ch.IsProductEnrolled(nil, nil, nil))
	assert.True(t, ch.IsProductEnrolled(strPtr("cat-1"), []string{"summer"}, nil))
}

func TestIsProductEnrolled_manualMode_enrollsNothingByDefault(t *testing.T) {
	ch := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeManual}

	assert.False(t, ch.IsProductEnrolled(strPtr("cat-1"), []string{"summer"}, nil))
}

func TestIsProductEnrolled_categoryMode_matchesOnlySameCategory(t *testing.T) {
	ch := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeCategory, EnrollmentCategoryID: strPtr("cat-1")}

	assert.True(t, ch.IsProductEnrolled(strPtr("cat-1"), nil, nil))
	assert.False(t, ch.IsProductEnrolled(strPtr("cat-2"), nil, nil))
	assert.False(t, ch.IsProductEnrolled(nil, nil, nil), "product with no category never matches a category rule")
}

func TestIsProductEnrolled_tagMode_matchesOnlyProductsWithThatTag(t *testing.T) {
	ch := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeTag, EnrollmentTag: "summer"}

	assert.True(t, ch.IsProductEnrolled(nil, []string{"summer", "sale"}, nil))
	assert.False(t, ch.IsProductEnrolled(nil, []string{"winter"}, nil))
	assert.False(t, ch.IsProductEnrolled(nil, nil, nil))
}

func TestIsProductEnrolled_includeOverride_winsOverManualMode(t *testing.T) {
	ch := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeManual}
	override := &channels.SalesChannelProductOverride{Included: true}

	assert.True(t, ch.IsProductEnrolled(nil, nil, override))
}

func TestIsProductEnrolled_excludeOverride_winsOverAllMode(t *testing.T) {
	ch := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeAll}
	override := &channels.SalesChannelProductOverride{Included: false}

	assert.False(t, ch.IsProductEnrolled(strPtr("cat-1"), []string{"summer"}, override))
}

func TestIsProductEnrolled_excludeOverride_winsOverMatchingCategoryOrTagRule(t *testing.T) {
	categoryCh := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeCategory, EnrollmentCategoryID: strPtr("cat-1")}
	tagCh := channels.SalesChannel{EnrollmentMode: channels.EnrollmentModeTag, EnrollmentTag: "summer"}
	exclude := &channels.SalesChannelProductOverride{Included: false}

	assert.False(t, categoryCh.IsProductEnrolled(strPtr("cat-1"), nil, exclude))
	assert.False(t, tagCh.IsProductEnrolled(nil, []string{"summer"}, exclude))
}
