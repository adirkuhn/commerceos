package pos

import "context"

type ConnectionRepository interface {
	Create(ctx context.Context, c *Connection) error
	FindByID(ctx context.Context, id string) (*Connection, error)
	// FindByAPIKeyHash is used by the POS API-key auth middleware to
	// resolve which tenant/connection a request belongs to -- called
	// before any tenant is known, so (like
	// iam.ScannerPairingCodeRepository.FindPendingByCode) it must run
	// under tenant.WithSuperAdmin (bypass_rls). Only ever matches an
	// 'active' connection.
	FindByAPIKeyHash(ctx context.Context, apiKeyHash string) (*Connection, error)
	// List returns every connection for the current tenant (RLS-scoped,
	// like every other List method in this codebase) -- callers filter by
	// Status in Go where needed (e.g. the pending-approvals view).
	List(ctx context.Context) ([]Connection, error)
	Update(ctx context.Context, c *Connection) error
}
