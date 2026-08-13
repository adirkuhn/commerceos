package pos

import "time"

type ConnectionStatus string

const (
	ConnectionStatusPending ConnectionStatus = "pending"
	ConnectionStatusActive  ConnectionStatus = "active"
	ConnectionStatusRevoked ConnectionStatus = "revoked"
)

// Connection tracks one external POS desktop's integration with a tenant --
// bootstrapped by presenting an lm2-signed license (see
// infrastructure/lm2license), left 'pending' until a human explicitly
// approves it (application/pos.ApprovePOSConnectionUseCase), only then
// minting an API key. A leaked license alone can get a connection to
// 'pending' -- it can never reach 'active' (and therefore never gets real
// API access) without that explicit approval step. See CLAUDE.md/session
// notes for the full threat-model reasoning.
type Connection struct {
	ID       string
	TenantID string
	// DeviceID is self-reported by the POS's license (device_binding) --
	// audit/display only, never used as an access-control check (it can't
	// be independently verified over HTTP).
	DeviceID string
	Status   ConnectionStatus
	// APIKeyHash is the SHA-256 hex digest of the minted API key -- the raw
	// key is never persisted, only ever returned once via
	// GetPOSConnectionStatusUseCase before KeyAcknowledged flips true.
	APIKeyHash *string
	// LicenseID/LicenseFingerprint are copied from the verified license
	// payload for audit purposes -- not used for access control after
	// bootstrap (the minted APIKeyHash is the sole ongoing credential).
	LicenseID          string
	LicenseFingerprint string
	// KeyAcknowledged is set by the POS after it confirms it received and
	// stored the raw API key -- GetPOSConnectionStatusUseCase never reveals
	// the raw key again once true, even on a later poll.
	KeyAcknowledged bool
	RequestedAt     time.Time
	ApprovedAt      *time.Time
	// ApprovedBy is the approving lm2 user's ID -- plain string, not a FK,
	// same convention as every other *By field in this codebase.
	ApprovedBy string
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
