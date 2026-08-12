package iam

import "time"

type ScannerPairingCodeStatus string

const (
	ScannerPairingCodeStatusPending ScannerPairingCodeStatus = "pending"
	ScannerPairingCodeStatusClaimed ScannerPairingCodeStatus = "claimed"
)

// ScannerPairingCode is a short-lived, single-use code a desktop session
// generates so a handheld scanner can join the same authenticated identity
// without its own login flow. UserID/Role are captured from the desktop
// caller's JWT claims at generation time -- claiming the code mints a new
// token with the exact same claims shape (see
// application/iam/claim_pairing_code.go), so the scanner acts as the same
// user through the existing JWT middleware, unchanged. Role is a plain
// string, not middleware.Role -- domain/ must have zero imports from
// infrastructure/, so the mapping happens at the handler boundary.
type ScannerPairingCode struct {
	ID       string
	TenantID string
	// Code is the short (6-digit) code displayed on the desktop and typed on
	// the handheld. Guessable within its keyspace (1e6) -- the claim
	// endpoint for this path is rate-limited (see
	// infrastructure/middleware.IPRateLimiter). Unique only among pending
	// codes (see migration) -- once claimed or expired the text can be
	// reused.
	Code string
	// Token is a high-entropy opaque value (128-bit random, base64url) that
	// QR codes and barcodes encode for the handheld's camera/scan-engine to
	// read -- unguessable within the pairing TTL, so its claim path carries
	// no rate-limit dependency. Same pending-uniqueness rule as Code.
	Token     string
	UserID    string
	Role      string
	Status    ScannerPairingCodeStatus
	ExpiresAt time.Time
	CreatedAt time.Time
	ClaimedAt *time.Time
}
