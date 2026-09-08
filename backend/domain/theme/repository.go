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
