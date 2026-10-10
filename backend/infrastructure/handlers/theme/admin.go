package theme

import (
	"net/http"

	"github.com/gin-gonic/gin"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func pageParams(page *gentheme.Page, perPage *gentheme.PerPage) (int, int) {
	p, pp := 1, 20
	if page != nil {
		p = *page
	}
	if perPage != nil {
		pp = *perPage
	}
	return p, pp
}

func (h *Handler) ListThemes(c *gin.Context, params gentheme.ListThemesParams) {
	page, perPage := pageParams(params.Page, params.PerPage)
	list, total, err := h.uc.ListThemes.Execute(c.Request.Context(), params.SalesChannelId.String(), page, perPage)
	if fail(c, err) {
		return
	}
	data := make([]gentheme.Theme, len(list))
	for i, t := range list {
		data[i] = toGenTheme(t)
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, gentheme.ThemeListResponse{Data: &data, Total: &totalInt})
}

func (h *Handler) CreateTheme(c *gin.Context) {
	var req gentheme.CreateThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	preset := theme.PresetDefault
	if req.PresetKey != nil {
		preset = string(*req.PresetKey)
	}
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	th, err := h.uc.CreateTheme.Execute(c.Request.Context(), apptheme.CreateThemeInput{
		SalesChannelID: req.SalesChannelId.String(), Name: req.Name, PresetKey: preset, UserID: userID,
	})
	if fail(c, err) {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenTheme(*th)})
}

func (h *Handler) GetTheme(c *gin.Context, themeID gentheme.ThemeId) {
	d, err := h.uc.GetTheme.Execute(c.Request.Context(), themeID.String())
	if fail(c, err) {
		return
	}
	middleware.OK(c, gentheme.ThemeDetail{
		Theme: toGenTheme(d.Theme), Draft: toGenSummaryPtr(d.Draft), Published: toGenSummaryPtr(d.Published),
	})
}

func (h *Handler) RenameTheme(c *gin.Context, themeID gentheme.ThemeId) {
	var req gentheme.RenameThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	th, err := h.uc.RenameTheme.Execute(c.Request.Context(), themeID.String(), req.Name)
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGenTheme(*th))
}

func (h *Handler) DeleteTheme(c *gin.Context, themeID gentheme.ThemeId) {
	if fail(c, h.uc.DeleteTheme.Execute(c.Request.Context(), themeID.String())) {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListThemeVersions(c *gin.Context, themeID gentheme.ThemeId, params gentheme.ListThemeVersionsParams) {
	page, perPage := pageParams(params.Page, params.PerPage)
	list, total, err := h.uc.ListVersions.Execute(c.Request.Context(), themeID.String(), page, perPage)
	if fail(c, err) {
		return
	}
	data := make([]gentheme.VersionSummary, len(list))
	for i, v := range list {
		data[i] = toGenSummary(v)
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, gentheme.VersionListResponse{Data: &data, Total: &totalInt})
}

func (h *Handler) PublishTheme(c *gin.Context, themeID gentheme.ThemeId) {
	h.publish(c, themeID.String())
}

func (h *Handler) publish(c *gin.Context, themeID string) {
	var req gentheme.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	note := ""
	if req.Note != nil {
		note = *req.Note
	}
	v, err := h.uc.PublishTheme.Execute(c.Request.Context(), apptheme.PublishThemeInput{
		ThemeID: themeID, ExpectedRevision: req.ExpectedRevision, Note: note,
	})
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGenVersion(*v))
}

func (h *Handler) RollbackTheme(c *gin.Context, themeID gentheme.ThemeId) {
	var req gentheme.RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	th, err := h.uc.RollbackTheme.Execute(c.Request.Context(), themeID.String(), req.VersionId.String())
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGenTheme(*th))
}

func (h *Handler) CreateThemeSession(c *gin.Context, themeID gentheme.ThemeId) {
	var req gentheme.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	claims := middleware.ClaimsFromGin(c)
	if claims == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "no claims in context")
		return
	}
	h.createSession(c, apptheme.CreateSessionInput{
		ThemeID: themeID.String(), UserID: claims.UserID, Role: string(claims.Role), Purpose: theme.SessionPurpose(req.Purpose),
		Locale: sessionLocale(req.Locale),
	})
}

func (h *Handler) createSession(c *gin.Context, in apptheme.CreateSessionInput) {
	tok, err := h.uc.CreateSession.Execute(c.Request.Context(), in)
	if fail(c, err) {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: gentheme.SessionToken{Token: tok.Token, ExpiresAt: tok.ExpiresAt}})
}

func sessionLocale(l *gentheme.Locale) locale.Locale {
	if l == nil {
		return ""
	}
	return locale.Locale(*l)
}
