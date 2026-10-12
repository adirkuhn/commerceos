package messagecenter

import (
	"context"
	"time"
)

// SetupRepository remembers that a tenant was given the default templates
// and rules, so defaults the tenant deletes later don't come back.
type SetupRepository interface {
	// Claim records the tenant in ctx as seeded and reports whether this
	// call did it -- false when it already was, also when a concurrent
	// transaction claimed it first, so only one caller seeds.
	Claim(ctx context.Context, at time.Time) (bool, error)
	// ListUnseededTenants lists tenants (across RLS -- needs a super-admin
	// context) that have not been given the defaults yet.
	ListUnseededTenants(ctx context.Context) ([]string, error)
}
