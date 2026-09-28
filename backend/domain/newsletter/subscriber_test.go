package newsletter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/newsletter"
)

var now = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func TestNormalizeEmail(t *testing.T) {
	e, err := newsletter.NormalizeEmail("  Ann@Example.COM ")
	require.NoError(t, err)
	assert.Equal(t, "ann@example.com", e)

	for _, bad := range []string{"", "ann", "ann@", "Ann <ann@example.com>", strings.Repeat("a", 250) + "@x.io", "a b@x.io"} {
		_, err := newsletter.NormalizeEmail(bad)
		assert.ErrorIs(t, err, newsletter.ErrInvalidEmail, bad)
	}
}

func TestNewSubscriber_isPendingUntilConfirmed(t *testing.T) {
	s := newsletter.NewSubscriber("tenant-1", "chan-1", "ann@example.com", "confirm-raw", "unsub-raw", now)

	assert.Equal(t, newsletter.StatusPending, s.Status)
	assert.Equal(t, newsletter.HashToken("confirm-raw"), s.ConfirmTokenHash)
	assert.NotEqual(t, "confirm-raw", s.ConfirmTokenHash)
	assert.Equal(t, "unsub-raw", s.UnsubscribeToken)
	assert.Equal(t, now, s.ConsentAt)
	assert.Equal(t, now, s.ConfirmationSentAt)
}

func TestConfirm(t *testing.T) {
	s := newsletter.NewSubscriber("t", "c", "a@x.io", "tok", "u", now)

	assert.ErrorIs(t, s.Confirm(now.Add(newsletter.ConfirmationTTL+time.Second)), newsletter.ErrInvalidToken, "expired")
	require.NoError(t, s.Confirm(now.Add(time.Hour)))
	assert.Equal(t, newsletter.StatusConfirmed, s.Status)
	require.NotNil(t, s.ConfirmedAt)
	assert.Empty(t, s.ConfirmTokenHash, "a confirmation link works once")
	assert.ErrorIs(t, s.Confirm(now.Add(2*time.Hour)), newsletter.ErrInvalidToken)
}

func TestResubscribe(t *testing.T) {
	s := newsletter.NewSubscriber("t", "c", "a@x.io", "tok", "u", now)

	assert.False(t, s.Resubscribe("tok2", now.Add(time.Minute)), "pending: resend throttled")
	assert.True(t, s.Resubscribe("tok2", now.Add(newsletter.ResendAfter)), "pending: resend after the throttle")
	assert.Equal(t, newsletter.HashToken("tok2"), s.ConfirmTokenHash)

	require.NoError(t, s.Confirm(now.Add(newsletter.ResendAfter)))
	assert.False(t, s.Resubscribe("tok3", now.Add(48*time.Hour)), "already confirmed: nothing to send")

	s.Unsubscribe(now.Add(72 * time.Hour))
	assert.Equal(t, newsletter.StatusUnsubscribed, s.Status)
	assert.True(t, s.Resubscribe("tok4", now.Add(96*time.Hour)), "unsubscribed can opt in again, via a new confirmation")
	assert.Equal(t, newsletter.StatusPending, s.Status)
	assert.Equal(t, now.Add(96*time.Hour), s.ConsentAt)
}
