// Package theme serves the storefront theme API (api/theme): the admin
// theme library behind the merchant JWT, and -- on the public router -- the
// storefront-hosted editor (edit-session token) and the public storefront
// reads (channel storefront key).
package theme

import (
	"context"

	"github.com/gin-gonic/gin"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	apptheme "github.com/adirkuhn/commerceos/application/theme"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// UseCases lists the ports the handler calls, one small interface each.
type UseCases struct {
	ListThemes interface {
		Execute(ctx context.Context, salesChannelID string, page, perPage int) ([]theme.Theme, int64, error)
	}
	CreateTheme interface {
		Execute(ctx context.Context, in apptheme.CreateThemeInput) (*theme.Theme, error)
	}
	GetTheme interface {
		Execute(ctx context.Context, id string) (*apptheme.ThemeDetail, error)
	}
	RenameTheme interface {
		Execute(ctx context.Context, id, name string) (*theme.Theme, error)
	}
	DeleteTheme interface {
		Execute(ctx context.Context, id string) error
	}
	ListVersions interface {
		Execute(ctx context.Context, themeID string, page, perPage int) ([]theme.Version, int64, error)
	}
	PublishTheme interface {
		Execute(ctx context.Context, in apptheme.PublishThemeInput) (*theme.Version, error)
	}
	RollbackTheme interface {
		Execute(ctx context.Context, themeID, versionID string) (*theme.Theme, error)
	}
	CreateSession interface {
		Execute(ctx context.Context, in apptheme.CreateSessionInput) (*apptheme.SessionToken, error)
	}
	ResolveSession interface {
		Execute(ctx context.Context, rawToken string, purpose theme.SessionPurpose) (*theme.Session, error)
	}
	OpenDraft interface {
		Execute(ctx context.Context, themeID, userID string) (*apptheme.VersionView, error)
	}
	SaveDraftSettings interface {
		Execute(ctx context.Context, in apptheme.SaveDraftSettingsInput) (*theme.Version, error)
	}
	GetDraftDocument interface {
		Execute(ctx context.Context, themeID string, kind theme.DocumentKind, key string) (*apptheme.DocumentView, error)
	}
	SaveDraftDocument interface {
		Execute(ctx context.Context, in apptheme.SaveDraftDocumentInput) (*apptheme.DraftWrite, error)
	}
	CreatePage interface {
		Execute(ctx context.Context, in apptheme.CreatePageInput) (*apptheme.DraftWrite, error)
	}
	DeletePage interface {
		Execute(ctx context.Context, themeID string, expectedRevision int, key string) (int, error)
	}
	GetLiveTheme interface {
		Execute(ctx context.Context, salesChannelID string) (*apptheme.VersionView, error)
	}
	GetPublishedDocument interface {
		Execute(ctx context.Context, salesChannelID, versionID string, kind theme.DocumentKind, key string) (*apptheme.DocumentView, error)
	}
	GetPreviewTheme interface {
		Execute(ctx context.Context, salesChannelID, themeID string) (*apptheme.VersionView, error)
	}
	GetPreviewDocument interface {
		Execute(ctx context.Context, salesChannelID, themeID string, kind theme.DocumentKind, key string) (*apptheme.DocumentView, error)
	}
	ListAssets interface {
		Execute(ctx context.Context, page, perPage int) ([]theme.Asset, int64, error)
	}
	UploadAsset interface {
		Execute(ctx context.Context, in apptheme.UploadAssetInput) (*theme.Asset, error)
	}
	UpdateAsset interface {
		Execute(ctx context.Context, id, alt string) (*theme.Asset, error)
	}
	DeleteAsset interface {
		Execute(ctx context.Context, id string) error
	}
	ResolveStorefrontKey interface {
		Execute(ctx context.Context, plaintextKey string) (apikeysDomain.ResolvedAPIKey, error)
	}
}

type Handler struct{ uc UseCases }

func NewHandler(uc UseCases) *Handler { return &Handler{uc: uc} }

var _ gentheme.ServerInterface = (*Handler)(nil)

func wrapper(h *Handler) gentheme.ServerInterfaceWrapper {
	return gentheme.ServerInterfaceWrapper{
		Handler: h,
		ErrorHandler: func(c *gin.Context, err error, status int) {
			middleware.Fail(c, status, "VALIDATION_ERROR", err.Error())
		},
	}
}

// RegisterAdmin mounts the theme library on a JWT-protected group. Reads
// are open to every role; writes (including minting sessions) need
// owner/admin.
func RegisterAdmin(rg gin.IRouter, h *Handler) {
	w := wrapper(h)
	g := rg.Group("")
	g.Use(middleware.RequireRoleForWrites(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	g.GET("/theme/themes", w.ListThemes)
	g.POST("/theme/themes", w.CreateTheme)
	g.GET("/theme/themes/:themeId", w.GetTheme)
	g.PATCH("/theme/themes/:themeId", w.RenameTheme)
	g.DELETE("/theme/themes/:themeId", w.DeleteTheme)
	g.GET("/theme/themes/:themeId/versions", w.ListThemeVersions)
	g.POST("/theme/themes/:themeId/publish", w.PublishTheme)
	g.POST("/theme/themes/:themeId/rollback", w.RollbackTheme)
	g.POST("/theme/themes/:themeId/sessions", w.CreateThemeSession)
}

// RegisterPublic mounts the editor and storefront routes on a group with
// no JWT: each authenticates its own bearer credential.
func RegisterPublic(rg gin.IRouter, h *Handler) {
	w := wrapper(h)
	rg.POST("/theme-editor/draft", w.OpenDraft)
	rg.PUT("/theme-editor/draft/settings", w.SaveDraftSettings)
	rg.GET("/theme-editor/draft/documents/:kind/:key", w.GetDraftDocument)
	rg.PUT("/theme-editor/draft/documents/:kind/:key", w.SaveDraftDocument)
	rg.POST("/theme-editor/draft/pages", w.CreateDraftPage)
	rg.DELETE("/theme-editor/draft/pages/:key", w.DeleteDraftPage)
	rg.POST("/theme-editor/publish", w.PublishDraft)
	rg.POST("/theme-editor/preview-sessions", w.CreatePreviewSession)
	rg.GET("/theme-editor/assets", w.ListAssets)
	rg.POST("/theme-editor/assets", w.UploadAsset)
	rg.PATCH("/theme-editor/assets/:assetId", w.UpdateAsset)
	rg.DELETE("/theme-editor/assets/:assetId", w.DeleteAsset)
	rg.GET("/storefront/v1/theme", w.GetStorefrontTheme)
	rg.GET("/storefront/v1/theme/versions/:versionId/documents/:kind/:key", w.GetStorefrontThemeDocument)
	rg.GET("/storefront/v1/theme/preview/documents/:kind/:key", w.GetStorefrontPreviewDocument)
}
