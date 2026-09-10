package theme

import "context"

// Repository is scoped to the current tenant via RLS (set by
// tenant.WithTenant), same convention as every other repository in this
// codebase.
type Repository interface {
	// FindBySlug looks up one channel's page by slug. Nil (not an error)
	// when none exists yet -- a channel with no saved theme has no row at
	// all, not a zero-value one (see application/theme's default-page
	// fallback for what a caller does with nil).
	FindBySlug(ctx context.Context, salesChannelID string, slug PageSlug) (*ThemePage, error)
	// Upsert replaces the whole page document for (SalesChannelID, Slug) --
	// a page's block list is always saved as one unit from the canvas
	// editor, never patched incrementally (see ThemePage's own doc
	// comment).
	Upsert(ctx context.Context, p *ThemePage) error
}

// EditSessionRepository persists EditSession rows -- scoped to the current
// tenant via RLS (set by tenant.WithTenant) for Create, and looked up
// under a super-admin bypass for FindByToken (see
// application/theme.ResolveEditSessionUseCase's own doc comment on why: a
// caller presenting a token has no tenant identity yet, that's exactly
// what a successful resolution establishes -- same cross-tenant-read-by-
// secret pattern application/apikeys.ResolveAPIKeyUseCase already uses).
type EditSessionRepository interface {
	Create(ctx context.Context, s *EditSession) error
	// FindByToken returns nil, nil (not an error) when no session matches
	// -- callers treat "no row" and "row exists but expired" identically
	// (see EditSession.IsExpired), never leaking which to the caller.
	FindByToken(ctx context.Context, token string) (*EditSession, error)
}
