package notifications_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/notifications"
)

var now = time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)

func TestCategoryOf_groupsTypesByPrefix(t *testing.T) {
	assert.Equal(t, notifications.CategoryImports, notifications.CategoryOf(notifications.TypeImportFailed))
	assert.Equal(t, notifications.CategoryPurchaseDocuments, notifications.CategoryOf(notifications.TypeReceivingDiscrepanciesFound))
	assert.Equal(t, notifications.CategoryChannelOffers, notifications.CategoryOf(notifications.TypeBolNotForSale))
	assert.Equal(t, notifications.CategoryOther, notifications.CategoryOf("something.new"))
}

func TestDefaultPreferences_inAppOnEmailOff(t *testing.T) {
	p := notifications.DefaultPreferences("tenant-1")

	for _, c := range notifications.Categories {
		assert.Equal(t, notifications.ChannelPrefs{InApp: true, Email: false}, p.For(c), c)
	}
	assert.Empty(t, p.EmailRecipients)
}

func TestPreferences_ForType(t *testing.T) {
	p, err := notifications.NewPreferences("tenant-1", notifications.PreferencesInput{
		Categories:      map[notifications.Category]notifications.ChannelPrefs{notifications.CategoryImports: {InApp: false, Email: true}},
		EmailRecipients: []string{"ops@example.nl"},
	}, now)
	require.NoError(t, err)

	assert.Equal(t, notifications.ChannelPrefs{InApp: false, Email: true}, p.ForType(notifications.TypeImportCompleted))
	// categories not mentioned keep the defaults
	assert.Equal(t, notifications.ChannelPrefs{InApp: true, Email: false}, p.ForType(notifications.TypeBolNotForSale))
}

func TestNewPreferences_normalizesRecipients(t *testing.T) {
	p, err := notifications.NewPreferences("tenant-1", notifications.PreferencesInput{
		EmailRecipients: []string{" Ops@Example.nl ", "ops@example.nl", "", "boss@example.nl"},
	}, now)

	require.NoError(t, err)
	assert.Equal(t, []string{"Ops@Example.nl", "boss@example.nl"}, p.EmailRecipients)
	assert.Equal(t, now, p.UpdatedAt)
}

func TestNewPreferences_rejects(t *testing.T) {
	cases := []struct {
		name string
		in   notifications.PreferencesInput
		want error
	}{
		{"invalid recipient", notifications.PreferencesInput{EmailRecipients: []string{"nope"}}, notifications.ErrInvalidRecipient},
		{"too many recipients", notifications.PreferencesInput{EmailRecipients: manyRecipients(11)}, notifications.ErrTooManyRecipients},
		{"email on without recipients", notifications.PreferencesInput{
			Categories: map[notifications.Category]notifications.ChannelPrefs{notifications.CategoryImports: {Email: true}},
		}, notifications.ErrEmailRecipientsRequired},
		{"unknown category", notifications.PreferencesInput{
			Categories: map[notifications.Category]notifications.ChannelPrefs{"billing": {InApp: true}},
		}, notifications.ErrUnknownCategory},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := notifications.NewPreferences("tenant-1", tc.in, now)
			assert.True(t, errors.Is(err, tc.want), "got %v", err)
		})
	}
}

func manyRecipients(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = string(rune('a'+i)) + "@example.nl"
	}
	return out
}
