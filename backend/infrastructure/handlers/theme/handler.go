// Package theme wires the admin-only (JWT) Theme Builder API (see
// api/theme's doc comment) to its use cases.
package theme

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	appTheme "github.com/adirkuhn/commerceos/application/theme"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type getThemePageUC interface {
	Execute(ctx context.Context, salesChannelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error)
}

type saveThemePageUC interface {
	Execute(ctx context.Context, in appTheme.SaveThemePageInput) (*themeDomain.ThemePage, error)
}

type createEditSessionUC interface {
	Execute(ctx context.Context, salesChannelID string) (*themeDomain.EditSession, error)
}

type Handler struct {
	getThemePage      getThemePageUC
	saveThemePage     saveThemePageUC
	createEditSession createEditSessionUC
}

func NewHandler(getThemePage getThemePageUC, saveThemePage saveThemePageUC, createEditSession createEditSessionUC) *Handler {
	return &Handler{getThemePage: getThemePage, saveThemePage: saveThemePage, createEditSession: createEditSession}
}

var _ gentheme.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	themeRepo := repositories.NewThemePageRepo(db)
	channelRepo := repositories.NewSalesChannelRepo(db)
	editSessionRepo := repositories.NewThemeEditSessionRepo(db)
	h := NewHandler(
		appTheme.NewGetThemePageUseCase(db, themeRepo),
		appTheme.NewSaveThemePageUseCase(db, themeRepo, channelRepo),
		appTheme.NewCreateEditSessionUseCase(db, editSessionRepo, channelRepo),
	)
	gentheme.RegisterHandlers(rg, h)
}

func (h *Handler) CreateThemeEditSession(c *gin.Context, channelID string) {
	session, err := h.createEditSession.Execute(c.Request.Context(), channelID)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gentheme.EditSession{Token: session.Token, ExpiresAt: session.ExpiresAt})
}

func (h *Handler) GetThemePage(c *gin.Context, channelID string, slug gentheme.PageSlugParam) {
	page, err := h.getThemePage.Execute(c.Request.Context(), channelID, themeDomain.PageSlug(slug))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenThemePage(*page))
}

func (h *Handler) SaveThemePage(c *gin.Context, channelID string, slug gentheme.PageSlugParam) {
	var req gentheme.SaveThemePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	blocks := make([]themeDomain.Block, len(req.Blocks))
	for i, b := range req.Blocks {
		blocks[i] = toDomainBlock(b)
	}
	var puckData map[string]any
	if req.PuckData != nil {
		puckData = *req.PuckData
	}
	editorMode := themeDomain.EditorModeFreeform
	if req.EditorMode != nil {
		editorMode = themeDomain.EditorMode(*req.EditorMode)
	}
	page, err := h.saveThemePage.Execute(c.Request.Context(), appTheme.SaveThemePageInput{
		SalesChannelID: channelID, Slug: themeDomain.PageSlug(slug), CanvasWidth: req.CanvasWidth, Blocks: blocks, CustomCSS: req.CustomCss,
		EditorMode: editorMode, PuckData: puckData,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenThemePage(*page))
}

func toGenThemePage(p themeDomain.ThemePage) gentheme.ThemePage {
	blocks := make([]gentheme.Block, len(p.Blocks))
	for i, b := range p.Blocks {
		blocks[i] = toGenBlock(b)
	}
	themePage := gentheme.ThemePage{
		SalesChannelId: p.SalesChannelID, Slug: gentheme.PageSlug(p.Slug), CanvasWidth: p.CanvasWidth, Blocks: blocks, CustomCss: p.CustomCSS,
		EditorMode: gentheme.EditorMode(p.EditorMode),
	}
	if p.PuckData != nil {
		themePage.PuckData = &p.PuckData
	}
	return themePage
}

func toGenBlock(b themeDomain.Block) gentheme.Block {
	layout := gentheme.BlockLayout{Desktop: gentheme.Rect{
		X: b.Layout.Desktop.X, Y: b.Layout.Desktop.Y, Width: b.Layout.Desktop.Width, Height: b.Layout.Desktop.Height,
	}}
	if b.Layout.Mobile != nil {
		layout.Mobile = &gentheme.Rect{X: b.Layout.Mobile.X, Y: b.Layout.Mobile.Y, Width: b.Layout.Mobile.Width, Height: b.Layout.Mobile.Height}
	}
	props := b.Props
	if props == nil {
		props = map[string]any{}
	}
	return gentheme.Block{Id: b.ID, Type: gentheme.BlockType(b.Type), Layout: layout, Props: props}
}

func toDomainBlock(b gentheme.Block) themeDomain.Block {
	layout := themeDomain.BlockLayout{Desktop: themeDomain.Rect{
		X: b.Layout.Desktop.X, Y: b.Layout.Desktop.Y, Width: b.Layout.Desktop.Width, Height: b.Layout.Desktop.Height,
	}}
	if b.Layout.Mobile != nil {
		layout.Mobile = &themeDomain.Rect{X: b.Layout.Mobile.X, Y: b.Layout.Mobile.Y, Width: b.Layout.Mobile.Width, Height: b.Layout.Mobile.Height}
	}
	return themeDomain.Block{ID: b.Id, Type: themeDomain.BlockType(b.Type), Layout: layout, Props: b.Props}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, themeDomain.ErrInvalidBlockType), errors.Is(err, themeDomain.ErrInvalidPageSlug):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrSalesChannelNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("theme: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
