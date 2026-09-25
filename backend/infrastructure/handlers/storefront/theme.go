package storefront

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appTheme "github.com/adirkuhn/commerceos/application/theme"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (h *Handler) GetStorefrontTheme(c *gin.Context, slug genstorefront.StorefrontPageSlug) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	page, err := h.getStorefrontTheme.Execute(ctx, *resolved.SalesChannelID, string(slug))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenStorefrontThemePage(*page))
}

func toGenStorefrontThemePage(p themeDomain.ThemePage) genstorefront.StorefrontThemePage {
	blocks := make([]genstorefront.StorefrontBlock, len(p.Blocks))
	for i, b := range p.Blocks {
		blocks[i] = toGenStorefrontBlock(b)
	}
	page := genstorefront.StorefrontThemePage{
		Slug: genstorefront.StorefrontPageSlug(p.Slug), CanvasWidth: p.CanvasWidth, Blocks: blocks, CustomCss: p.CustomCSS,
		EditorMode: genstorefront.StorefrontEditorMode(p.EditorMode),
	}
	if p.PuckData != nil {
		page.PuckData = &p.PuckData
	}
	return page
}

func toGenStorefrontBlock(b themeDomain.Block) genstorefront.StorefrontBlock {
	layout := genstorefront.StorefrontBlockLayout{Desktop: genstorefront.StorefrontRect{
		X: b.Layout.Desktop.X, Y: b.Layout.Desktop.Y, Width: b.Layout.Desktop.Width, Height: b.Layout.Desktop.Height,
	}}
	if b.Layout.Mobile != nil {
		layout.Mobile = &genstorefront.StorefrontRect{X: b.Layout.Mobile.X, Y: b.Layout.Mobile.Y, Width: b.Layout.Mobile.Width, Height: b.Layout.Mobile.Height}
	}
	props := b.Props
	if props == nil {
		props = map[string]any{}
	}
	return genstorefront.StorefrontBlock{Id: b.ID, Type: genstorefront.StorefrontBlockType(b.Type), Layout: layout, Props: props}
}

func (h *Handler) GetStorefrontThemeDraft(c *gin.Context, slug genstorefront.StorefrontPageSlug) {
	resolved, ok := h.authenticateEditSession(c)
	if !ok {
		return
	}
	pageSlug := themeDomain.PageSlug(slug)
	if !pageSlug.Valid() {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", themeDomain.ErrInvalidPageSlug.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	page, err := h.getThemeDraft.Execute(ctx, resolved.SalesChannelID, pageSlug)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenStorefrontThemeDraftPage(*page, resolved.SalesChannelID))
}

func (h *Handler) SaveStorefrontThemeDraft(c *gin.Context, slug genstorefront.StorefrontPageSlug) {
	resolved, ok := h.authenticateEditSession(c)
	if !ok {
		return
	}
	var req genstorefront.SaveStorefrontThemeDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	blocks := make([]themeDomain.Block, len(req.Blocks))
	for i, b := range req.Blocks {
		blocks[i] = fromGenStorefrontBlock(b)
	}
	editorMode := themeDomain.EditorModeFreeform
	if req.EditorMode != nil {
		editorMode = themeDomain.EditorMode(*req.EditorMode)
	}
	var puckData map[string]any
	if req.PuckData != nil {
		puckData = *req.PuckData
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	page, err := h.saveThemeDraft.Execute(ctx, appTheme.SaveThemePageInput{
		SalesChannelID: resolved.SalesChannelID, Slug: themeDomain.PageSlug(slug), CanvasWidth: req.CanvasWidth,
		Blocks: blocks, CustomCSS: req.CustomCss, EditorMode: editorMode, PuckData: puckData,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenStorefrontThemeDraftPage(*page, resolved.SalesChannelID))
}

func toGenStorefrontThemeDraftPage(p themeDomain.ThemePage, salesChannelID string) genstorefront.StorefrontThemeDraftPage {
	blocks := make([]genstorefront.StorefrontBlock, len(p.Blocks))
	for i, b := range p.Blocks {
		blocks[i] = toGenStorefrontBlock(b)
	}
	draft := genstorefront.StorefrontThemeDraftPage{
		SalesChannelId: salesChannelID,
		Slug:           genstorefront.StorefrontPageSlug(p.Slug), CanvasWidth: p.CanvasWidth, Blocks: blocks, CustomCss: p.CustomCSS,
		EditorMode: genstorefront.StorefrontEditorMode(p.EditorMode),
	}
	if p.PuckData != nil {
		draft.PuckData = &p.PuckData
	}
	return draft
}

func fromGenStorefrontBlock(b genstorefront.StorefrontBlock) themeDomain.Block {
	layout := themeDomain.BlockLayout{Desktop: themeDomain.Rect{
		X: b.Layout.Desktop.X, Y: b.Layout.Desktop.Y, Width: b.Layout.Desktop.Width, Height: b.Layout.Desktop.Height,
	}}
	if b.Layout.Mobile != nil {
		layout.Mobile = &themeDomain.Rect{X: b.Layout.Mobile.X, Y: b.Layout.Mobile.Y, Width: b.Layout.Mobile.Width, Height: b.Layout.Mobile.Height}
	}
	return themeDomain.Block{ID: b.Id, Type: themeDomain.BlockType(b.Type), Layout: layout, Props: b.Props}
}
