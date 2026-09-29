package theme_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestNewSession_storesOnlyTheHash(t *testing.T) {
	s, err := theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.PurposeEdit, "", "raw-token", now)
	require.NoError(t, err)

	assert.NotEqual(t, "raw-token", s.TokenHash)
	assert.Equal(t, theme.HashToken("raw-token"), s.TokenHash)
	assert.Equal(t, now.Add(theme.EditSessionTTL), s.ExpiresAt)
}

func TestNewSession_previewLastsLonger(t *testing.T) {
	s, err := theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.PurposePreview, "", "t", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(theme.PreviewSessionTTL), s.ExpiresAt)
}

func TestNewSession_rejectsUnknownPurpose(t *testing.T) {
	_, err := theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.SessionPurpose("admin"), "", "t", now)
	assert.ErrorIs(t, err, theme.ErrInvalidSessionPurpose)
}

func TestSession_Authorize(t *testing.T) {
	s, _ := theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.PurposeEdit, "", "t", now)

	assert.NoError(t, s.Authorize(theme.PurposeEdit, now.Add(time.Hour)))
	assert.ErrorIs(t, s.Authorize(theme.PurposePreview, now), theme.ErrSessionInvalid, "purposes are not interchangeable")
	assert.ErrorIs(t, s.Authorize(theme.PurposeEdit, s.ExpiresAt), theme.ErrSessionInvalid, "expired")

	revokedAt := now
	s.RevokedAt = &revokedAt
	assert.ErrorIs(t, s.Authorize(theme.PurposeEdit, now), theme.ErrSessionInvalid, "revoked")
}

func TestHashToken_isStableAndHex(t *testing.T) {
	assert.Equal(t, theme.HashToken("abc"), theme.HashToken("abc"))
	assert.Len(t, theme.HashToken("abc"), 64)
}

// The edit session remembers the merchant's language, so the editor (on
// another origin) speaks it without it travelling in the URL.
func TestNewSession_carriesTheMerchantLanguage(t *testing.T) {
	now := time.Now()
	s, err := theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.PurposeEdit, locale.Portuguese, "t", now)
	require.NoError(t, err)
	assert.Equal(t, locale.Portuguese, s.Locale)

	blank, err := theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.PurposeEdit, "", "t", now)
	require.NoError(t, err)
	assert.Equal(t, locale.English, blank.Locale)

	_, err = theme.NewSession("tenant-1", "theme-1", "user-1", "customer_admin", theme.PurposeEdit, "de", "t", now)
	assert.ErrorIs(t, err, locale.ErrUnsupported)
}
