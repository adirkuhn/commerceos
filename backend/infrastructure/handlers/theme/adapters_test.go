package theme_test

import (
	"context"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/domain/theme"
)

// Func adapters so tests can stub each use-case port with a closure.

type listFunc func(context.Context, string, int, int) ([]theme.Theme, int64, error)

func (f listFunc) Execute(ctx context.Context, ch string, p, pp int) ([]theme.Theme, int64, error) {
	return f(ctx, ch, p, pp)
}

type createFunc func(context.Context, apptheme.CreateThemeInput) (*theme.Theme, error)

func (f createFunc) Execute(ctx context.Context, in apptheme.CreateThemeInput) (*theme.Theme, error) {
	return f(ctx, in)
}

type getFunc func(context.Context, string) (*apptheme.ThemeDetail, error)

func (f getFunc) Execute(ctx context.Context, id string) (*apptheme.ThemeDetail, error) {
	return f(ctx, id)
}

type renameFunc func(context.Context, string, string) (*theme.Theme, error)

func (f renameFunc) Execute(ctx context.Context, id, name string) (*theme.Theme, error) {
	return f(ctx, id, name)
}

type deleteFunc func(context.Context, string) error

func (f deleteFunc) Execute(ctx context.Context, id string) error { return f(ctx, id) }

type versionsFunc func(context.Context, string, int, int) ([]theme.Version, int64, error)

func (f versionsFunc) Execute(ctx context.Context, id string, p, pp int) ([]theme.Version, int64, error) {
	return f(ctx, id, p, pp)
}

type publishFunc func(context.Context, apptheme.PublishThemeInput) (*theme.Version, error)

func (f publishFunc) Execute(ctx context.Context, in apptheme.PublishThemeInput) (*theme.Version, error) {
	return f(ctx, in)
}

type rollbackFunc func(context.Context, string, string) (*theme.Theme, error)

func (f rollbackFunc) Execute(ctx context.Context, id, v string) (*theme.Theme, error) {
	return f(ctx, id, v)
}

type sessionFunc func(context.Context, apptheme.CreateSessionInput) (*apptheme.SessionToken, error)

func (f sessionFunc) Execute(ctx context.Context, in apptheme.CreateSessionInput) (*apptheme.SessionToken, error) {
	return f(ctx, in)
}

type resolveFunc func(context.Context, string, theme.SessionPurpose) (*theme.Session, error)

func (f resolveFunc) Execute(ctx context.Context, tok string, p theme.SessionPurpose) (*theme.Session, error) {
	return f(ctx, tok, p)
}

type openFunc func(context.Context, string, string) (*apptheme.VersionView, error)

func (f openFunc) Execute(ctx context.Context, id, user string) (*apptheme.VersionView, error) {
	return f(ctx, id, user)
}

type settingsFunc func(context.Context, apptheme.SaveDraftSettingsInput) (*theme.Version, error)

func (f settingsFunc) Execute(ctx context.Context, in apptheme.SaveDraftSettingsInput) (*theme.Version, error) {
	return f(ctx, in)
}

type getDocFunc func(context.Context, string, theme.DocumentKind, string) (*apptheme.DocumentView, error)

func (f getDocFunc) Execute(ctx context.Context, id string, k theme.DocumentKind, key string) (*apptheme.DocumentView, error) {
	return f(ctx, id, k, key)
}

type saveDocFunc func(context.Context, apptheme.SaveDraftDocumentInput) (*apptheme.DraftWrite, error)

func (f saveDocFunc) Execute(ctx context.Context, in apptheme.SaveDraftDocumentInput) (*apptheme.DraftWrite, error) {
	return f(ctx, in)
}

type createPageFunc func(context.Context, apptheme.CreatePageInput) (*apptheme.DraftWrite, error)

func (f createPageFunc) Execute(ctx context.Context, in apptheme.CreatePageInput) (*apptheme.DraftWrite, error) {
	return f(ctx, in)
}

type deletePageFunc func(context.Context, string, int, string) (int, error)

func (f deletePageFunc) Execute(ctx context.Context, id string, rev int, key string) (int, error) {
	return f(ctx, id, rev, key)
}

type liveFunc func(context.Context, string) (*apptheme.VersionView, error)

func (f liveFunc) Execute(ctx context.Context, ch string) (*apptheme.VersionView, error) {
	return f(ctx, ch)
}

type publishedDocFunc func(context.Context, string, string, theme.DocumentKind, string) (*apptheme.DocumentView, error)

func (f publishedDocFunc) Execute(ctx context.Context, ch, v string, k theme.DocumentKind, key string) (*apptheme.DocumentView, error) {
	return f(ctx, ch, v, k, key)
}

type previewFunc func(context.Context, string, string) (*apptheme.VersionView, error)

func (f previewFunc) Execute(ctx context.Context, ch, id string) (*apptheme.VersionView, error) {
	return f(ctx, ch, id)
}

type previewDocFunc func(context.Context, string, string, theme.DocumentKind, string) (*apptheme.DocumentView, error)

func (f previewDocFunc) Execute(ctx context.Context, ch, id string, k theme.DocumentKind, key string) (*apptheme.DocumentView, error) {
	return f(ctx, ch, id, k, key)
}

type keyFunc func(context.Context, string) (apikeysDomain.ResolvedAPIKey, error)

func (f keyFunc) Execute(ctx context.Context, k string) (apikeysDomain.ResolvedAPIKey, error) {
	return f(ctx, k)
}

type listAssetsFunc func(context.Context, int, int) ([]theme.Asset, int64, error)

func (f listAssetsFunc) Execute(ctx context.Context, p, pp int) ([]theme.Asset, int64, error) {
	return f(ctx, p, pp)
}

type uploadAssetFunc func(context.Context, apptheme.UploadAssetInput) (*theme.Asset, error)

func (f uploadAssetFunc) Execute(ctx context.Context, in apptheme.UploadAssetInput) (*theme.Asset, error) {
	return f(ctx, in)
}

type updateAssetFunc func(context.Context, string, string) (*theme.Asset, error)

func (f updateAssetFunc) Execute(ctx context.Context, id, alt string) (*theme.Asset, error) {
	return f(ctx, id, alt)
}
