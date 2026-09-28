package theme

import "context"

// All repositories are tenant-scoped via RLS (tenant.WithTenant) and read
// their transaction from ctx, like every other repository here.

type ThemeRepository interface {
	// Create fills in t.ID.
	Create(ctx context.Context, t *Theme) error
	// FindByID returns ErrThemeNotFound when missing.
	FindByID(ctx context.Context, id string) (*Theme, error)
	// FindByIDForUpdate row-locks the theme until the transaction ends --
	// publish/rollback take it so two publishes of one theme serialize.
	FindByIDForUpdate(ctx context.Context, id string) (*Theme, error)
	// FindLiveByChannel returns nil, nil when the channel has no live theme.
	FindLiveByChannel(ctx context.Context, salesChannelID string) (*Theme, error)
	ListByChannel(ctx context.Context, salesChannelID string, limit, offset int) ([]Theme, int64, error)
	// Update persists Name, PublishedVersionID, IsLive and UpdatedAt.
	Update(ctx context.Context, t *Theme) error
	// ClearLive takes every other theme of the channel off live.
	ClearLive(ctx context.Context, salesChannelID, exceptThemeID string) error
	Delete(ctx context.Context, id string) error
}

type VersionRepository interface {
	// Create fills in v.ID.
	Create(ctx context.Context, v *Version) error
	// FindByID returns ErrVersionNotFound when missing.
	FindByID(ctx context.Context, id string) (*Version, error)
	// FindDraft returns nil, nil when the theme has no draft. forUpdate
	// row-locks it for the rest of the transaction.
	FindDraft(ctx context.Context, themeID string, forUpdate bool) (*Version, error)
	NextNumber(ctx context.Context, themeID string) (int, error)
	// Update persists State, Revision, Settings, Note and SealedAt. The
	// database refuses to change a sealed version.
	Update(ctx context.Context, v *Version) error
	ListByTheme(ctx context.Context, themeID string, limit, offset int) ([]Version, int64, error)
	// PruneSealed deletes the theme's sealed versions (and their documents)
	// beyond the newest `keep`, never protectID; returns how many.
	PruneSealed(ctx context.Context, themeID string, keep int, protectID string) (int64, error)
}

type DocumentRepository interface {
	// Create fills in d.ID.
	Create(ctx context.Context, d *Document) error
	// CopyAll copies every document of one version into another.
	CopyAll(ctx context.Context, fromVersionID, toVersionID string) error
	// Find returns ErrDocumentNotFound when missing.
	Find(ctx context.Context, versionID string, kind DocumentKind, key string) (*Document, error)
	// List returns the version's documents without their trees.
	List(ctx context.Context, versionID string) ([]Document, error)
	// Update persists Title, SEO and Root.
	Update(ctx context.Context, d *Document) error
	Delete(ctx context.Context, id string) error
}

type SessionRepository interface {
	// Create fills in s.ID.
	Create(ctx context.Context, s *Session) error
	// FindByTokenHash returns nil, nil when no session matches. Called
	// before the tenant is known, so it runs under tenant.WithSuperAdmin.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
}
