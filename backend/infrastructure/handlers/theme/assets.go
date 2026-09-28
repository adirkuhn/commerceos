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

// Asset routes act on the edit session's tenant library.

func (h *Handler) ListAssets(c *gin.Context, params gentheme.ListAssetsParams) {
	if _, ok := h.editSession(c); !ok {
		return
	}
	page, perPage := pageParams(params.Page, params.PerPage)
	list, total, err := h.uc.ListAssets.Execute(c.Request.Context(), page, perPage)
	if fail(c, err) {
		return
	}
	data := make([]gentheme.Asset, len(list))
	for i, a := range list {
		data[i] = toGenAsset(a)
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, gentheme.AssetListResponse{Data: &data, Total: &totalInt})
}

// uploadBodyLimit leaves room for the multipart envelope around a
// maximum-size image.
const uploadBodyLimit = theme.MaxAssetBytes + 1<<20

func (h *Handler) UploadAsset(c *gin.Context) {
	s, ok := h.editSession(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, uploadBodyLimit)
	fh, err := c.FormFile("file")
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "file is required (max 10 MB)")
		return
	}
	if fh.Size > theme.MaxAssetBytes {
		fail(c, theme.ErrAssetTooLarge)
		return
	}
	f, err := fh.Open()
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "could not read the upload")
		return
	}
	defer f.Close()
	a, err := h.uc.UploadAsset.Execute(c.Request.Context(), apptheme.UploadAssetInput{
		UserID: s.UserID, Filename: fh.Filename, Data: f, Alt: c.PostForm("alt"),
	})
	if fail(c, err) {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenAsset(*a)})
}

func (h *Handler) UpdateAsset(c *gin.Context, assetID uuid.UUID) {
	if _, ok := h.editSession(c); !ok {
		return
	}
	var req gentheme.UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	a, err := h.uc.UpdateAsset.Execute(c.Request.Context(), assetID.String(), req.Alt)
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGenAsset(*a))
}

func (h *Handler) DeleteAsset(c *gin.Context, assetID uuid.UUID) {
	if _, ok := h.editSession(c); !ok {
		return
	}
	if fail(c, h.uc.DeleteAsset.Execute(c.Request.Context(), assetID.String())) {
		return
	}
	middleware.OK(c, nil)
}
