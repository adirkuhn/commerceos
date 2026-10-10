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

// Editor routes act on the theme the edit session was minted for -- never
// on an id from the request, so a session can't reach another theme.

func (h *Handler) OpenDraft(c *gin.Context) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	view, err := h.uc.OpenDraft.Execute(c.Request.Context(), s.ThemeID, s.UserID)
	if fail(c, err) {
		return
	}
	draft := toGenDraft(*view)
	// Sessions from before languages existed read as English.
	lang, _ := locale.Resolve(s.Locale)
	if lang == "" {
		lang = locale.English
	}
	draft.Locale = gentheme.Locale(lang)
	middleware.OK(c, draft)
}

func (h *Handler) SaveDraftSettings(c *gin.Context) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	var req gentheme.SaveSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	v, err := h.uc.SaveDraftSettings.Execute(c.Request.Context(), apptheme.SaveDraftSettingsInput{
		ThemeID: s.ThemeID, ExpectedRevision: req.ExpectedRevision, Settings: fromGenSettings(req.Settings),
	})
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGenVersion(*v))
}

func (h *Handler) GetDraftDocument(c *gin.Context, kind gentheme.DocumentKindParam, key gentheme.DocumentKeyParam) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	d, err := h.uc.GetDraftDocument.Execute(c.Request.Context(), s.ThemeID, theme.DocumentKind(kind), key)
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGenDocumentView(*d))
}

func (h *Handler) SaveDraftDocument(c *gin.Context, kind gentheme.DocumentKindParam, key gentheme.DocumentKeyParam) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	var req gentheme.SaveDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	root, err := fromGenNode(req.Root)
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	title := ""
	if req.Title != nil {
		title = *req.Title
	}
	w, err := h.uc.SaveDraftDocument.Execute(c.Request.Context(), apptheme.SaveDraftDocumentInput{
		ThemeID: s.ThemeID, ExpectedRevision: req.ExpectedRevision, Kind: theme.DocumentKind(kind), Key: key,
		Title: title, SEO: fromGenSEO(req.Seo), Translations: fromGenDocumentTranslations(req.Translations), Root: root,
	})
	if fail(c, err) {
		return
	}
	middleware.OK(c, gentheme.DraftWrite{Document: toGenDocument(w.Document), Revision: w.Revision})
}

func (h *Handler) CreateDraftPage(c *gin.Context) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	var req gentheme.CreatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	w, err := h.uc.CreatePage.Execute(c.Request.Context(), apptheme.CreatePageInput{
		ThemeID: s.ThemeID, ExpectedRevision: req.ExpectedRevision, Key: req.Key, Title: req.Title,
	})
	if fail(c, err) {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: gentheme.DraftWrite{Document: toGenDocument(w.Document), Revision: w.Revision}})
}

func (h *Handler) DeleteDraftPage(c *gin.Context, key gentheme.DocumentKeyParam, params gentheme.DeleteDraftPageParams) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	rev, err := h.uc.DeletePage.Execute(c.Request.Context(), s.ThemeID, params.ExpectedRevision, key)
	if fail(c, err) {
		return
	}
	middleware.OK(c, gin.H{"revision": rev})
}

func (h *Handler) PublishDraft(c *gin.Context) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	h.publish(c, s.ThemeID)
}

func (h *Handler) CreatePreviewSession(c *gin.Context) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	h.createSession(c, apptheme.CreateSessionInput{ThemeID: s.ThemeID, UserID: s.UserID, Role: s.Role, Purpose: theme.PurposePreview})
}
