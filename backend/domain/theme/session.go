package theme

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type SessionPurpose string

const (
	// PurposeEdit lets the storefront-hosted editor read and write one
	// theme's draft (and publish it, if the minting user's role may).
	PurposeEdit SessionPurpose = "edit"
	// PurposePreview lets a browser render one theme's draft read-only on
	// the storefront, never touching what shoppers see.
	PurposePreview SessionPurpose = "preview"
)

const (
	EditSessionTTL    = 8 * time.Hour
	PreviewSessionTTL = 24 * time.Hour
)

// Session is a short-lived bearer credential scoped to one theme. Only the
// token's hash is stored, so a database leak can't be replayed.
type Session struct {
	ID        string
	TenantID  string
	ThemeID   string
	UserID    string
	Role      string
	Purpose   SessionPurpose
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func NewSession(tenantID, themeID, userID, role string, purpose SessionPurpose, rawToken string, now time.Time) (Session, error) {
	var ttl time.Duration
	switch purpose {
	case PurposeEdit:
		ttl = EditSessionTTL
	case PurposePreview:
		ttl = PreviewSessionTTL
	default:
		return Session{}, ErrInvalidSessionPurpose
	}
	return Session{
		TenantID: tenantID, ThemeID: themeID, UserID: userID, Role: role, Purpose: purpose,
		TokenHash: HashToken(rawToken), ExpiresAt: now.Add(ttl), CreatedAt: now,
	}, nil
}

// Authorize fails with the same error for every reason (wrong purpose,
// expired, revoked) so a caller learns nothing about which it was.
func (s Session) Authorize(purpose SessionPurpose, now time.Time) error {
	if s.Purpose != purpose || s.RevokedAt != nil || !now.Before(s.ExpiresAt) {
		return ErrSessionInvalid
	}
	return nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
