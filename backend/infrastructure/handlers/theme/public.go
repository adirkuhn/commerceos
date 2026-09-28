package theme

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

const (
	// The live index changes on publish, so it's revalidated often; its
	// ETag is the live version id.
	liveIndexCache = "public, max-age=30"
	// A published version's documents never change.
	immutableCache = "public, max-age=31536000, immutable"
	previewCache   = "no-store"
)

func (h *Handler) GetStorefrontTheme(c *gin.Context, params gentheme.GetStorefrontThemeParams) {
	previewToken := ""
	if params.XThemePreview != nil {
		previewToken = *params.XThemePreview
	}
	caller, ok := h.storefrontCaller(c, previewToken)
	if !ok {
		return
	}
	c.Header("Vary", "Authorization, X-Theme-Preview")
	var view *apptheme.VersionView
	var err error
	if caller.preview != nil {
		view, err = h.uc.GetPreviewTheme.Execute(c.Request.Context(), caller.channelID, caller.preview.ThemeID)
	} else {
		view, err = h.uc.GetLiveTheme.Execute(c.Request.Context(), caller.channelID)
	}
	if failPublic(c, err) {
		return
	}
	if caller.preview != nil {
		c.Header("Cache-Control", previewCache)
	} else {
		etag := `"` + view.Version.ID + `"`
		c.Header("ETag", etag)
		c.Header("Cache-Control", liveIndexCache)
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}
	middleware.OK(c, gentheme.StorefrontTheme{
		VersionId: toUUID(view.Version.ID), VersionNumber: view.Version.Number, Preview: caller.preview != nil,
		Settings: toGenSettings(view.Version.Settings), Documents: toGenSummaries(view.Documents),
	})
}

func (h *Handler) GetStorefrontThemeDocument(c *gin.Context, versionID uuid.UUID, kind gentheme.DocumentKindParam, key gentheme.DocumentKeyParam) {
	caller, ok := h.storefrontCaller(c, "")
	if !ok {
		return
	}
	d, err := h.uc.GetPublishedDocument.Execute(c.Request.Context(), caller.channelID, versionID.String(), theme.DocumentKind(kind), key)
	if failPublic(c, err) {
		return
	}
	c.Header("Cache-Control", immutableCache)
	middleware.OK(c, toGenDocument(*d))
}

func (h *Handler) GetStorefrontPreviewDocument(c *gin.Context, kind gentheme.DocumentKindParam, key gentheme.DocumentKeyParam, params gentheme.GetStorefrontPreviewDocumentParams) {
	caller, ok := h.storefrontCaller(c, params.XThemePreview)
	if !ok {
		return
	}
	if caller.preview == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing preview session")
		return
	}
	d, err := h.uc.GetPreviewDocument.Execute(c.Request.Context(), caller.channelID, caller.preview.ThemeID, theme.DocumentKind(kind), key)
	if failPublic(c, err) {
		return
	}
	c.Header("Cache-Control", previewCache)
	middleware.OK(c, toGenDocument(*d))
}
