package storefront

import (
	"testing"
	"time"
)

func TestMagicLinkToken_IsRedeemable(t *testing.T) {
	now := time.Now()
	usedAt := now.Add(-time.Minute)

	cases := []struct {
		name  string
		token MagicLinkToken
		want  bool
	}{
		{"fresh", MagicLinkToken{ExpiresAt: now.Add(time.Minute)}, true},
		{"expired", MagicLinkToken{ExpiresAt: now.Add(-time.Minute)}, false},
		{"used", MagicLinkToken{ExpiresAt: now.Add(time.Minute), UsedAt: &usedAt}, false},
	}
	for _, tc := range cases {
		if got := tc.token.IsRedeemable(now); got != tc.want {
			t.Errorf("%s: IsRedeemable() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCustomerSession_IsExpired(t *testing.T) {
	now := time.Now()

	if (CustomerSession{ExpiresAt: now.Add(time.Minute)}).IsExpired(now) {
		t.Error("expected a session expiring in the future to not be expired")
	}
	if !(CustomerSession{ExpiresAt: now.Add(-time.Minute)}).IsExpired(now) {
		t.Error("expected a session that expired in the past to be expired")
	}
}
