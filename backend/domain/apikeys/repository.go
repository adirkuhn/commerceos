package apikeys

import "context"

// Repository is scoped to the current tenant via RLS (set by
// tenant.WithTenant) for every method except FindActiveTenantIDByHash,
// which is the one place that must run without a tenant already known --
// see application/apikeys.ResolveAPIKeyUseCase's doc comment.
type Repository interface {
	// Create fills in k.ID and k.CreatedAt.
	Create(ctx context.Context, k *APIKey) error
	ListByTenant(ctx context.Context) ([]APIKey, error)
	// Revoke returns ErrNotFound if id doesn't belong to (or exist in) the
	// current tenant. Idempotent otherwise is not guaranteed -- revoking an
	// already-revoked key is left to the caller to avoid, not silently
	// accepted here.
	Revoke(ctx context.Context, id string) error
	// FindActiveTenantIDByHash resolves a key hash to its tenant (and, if
	// scoped, its sales channel), across every tenant, only when the key
	// exists and isn't revoked -- the caller has no tenant to scope by
	// yet, that's exactly what a successful resolution establishes.
	FindActiveTenantIDByHash(ctx context.Context, hash string) (ResolvedAPIKey, error)
}
