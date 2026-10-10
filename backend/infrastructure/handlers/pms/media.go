package pms

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Media handlers ────────────────────────────────────────────────────────────

func (h *Handler) ListProductMedia(c *gin.Context, id genpms.Id) {
	media, err := h.listProductMedia.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.Media, len(media))
	for i, m := range media {
		resp[i] = toGenMedia(m)
	}
	c.JSON(http.StatusOK, genpms.MediaListResponse{Data: &resp})
}

func (h *Handler) UploadProductMedia(c *gin.Context, id genpms.Id) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrFileRequired)
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	defer f.Close()

	media, err := h.uploadMedia.Execute(c.Request.Context(), apppms.UploadMediaInput{
		ProductID:   &id,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Data:        f,
		AltText:     c.PostForm("alt_text"),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenMedia(*media)
	c.JSON(http.StatusCreated, genpms.MediaResponse{Data: &resp})
}

func (h *Handler) UpdateMedia(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	media, err := h.updateMedia.Execute(c.Request.Context(), apppms.UpdateMediaInput{
		ID:      id,
		AltText: req.AltText,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenMedia(*media)
	c.JSON(http.StatusOK, genpms.MediaResponse{Data: &resp})
}

func (h *Handler) DeleteMedia(c *gin.Context, id genpms.Id) {
	err := h.deleteMedia.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func toGenMedia(m pmsDomain.Media) genpms.Media {
	return genpms.Media{
		Id:        m.ID,
		TenantId:  m.TenantID,
		ProductId: m.ProductID,
		VariantId: m.VariantID,
		Url:       m.URL,
		Type:      genpms.MediaType(m.Type),
		Position:  m.Position,
		AltText:   m.AltText,
		CreatedAt: m.CreatedAt,
	}
}
