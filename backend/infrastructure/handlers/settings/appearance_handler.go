package settings

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	gensettings "github.com/adirkuhn/commerceos/api/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type getAppearanceUC interface {
	Execute(ctx context.Context) (*settingsDomain.Appearance, error)
}
type updateAppearanceUC interface {
	Execute(ctx context.Context, in settingsDomain.AppearanceInput) (*settingsDomain.Appearance, error)
}

// AppearanceHandler serves /settings/appearance. Embedded in Handler so the
// generated settings ServerInterface stays one type. Read by every role;
// writes are guarded by RequireRoleForWrites on the settings route group.
type AppearanceHandler struct {
	get    getAppearanceUC
	update updateAppearanceUC
}

func NewAppearanceHandler(get getAppearanceUC, update updateAppearanceUC) *AppearanceHandler {
	return &AppearanceHandler{get: get, update: update}
}

func (h *AppearanceHandler) GetAppearance(c *gin.Context) {
	a, err := h.get.Execute(c.Request.Context())
	if err != nil {
		failAppearance(c, err)
		return
	}
	resp := toGenAppearance(*a)
	c.JSON(http.StatusOK, gensettings.AppearanceResponse{Data: &resp})
}

func (h *AppearanceHandler) UpdateAppearance(c *gin.Context) {
	var req gensettings.UpdateAppearanceJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	a, err := h.update.Execute(c.Request.Context(), settingsDomain.AppearanceInput{
		Font: string(req.Font), AccentColor: req.AccentColor, SidebarColor: req.SidebarColor,
		Corners: string(req.Corners), TextSize: string(req.TextSize), Density: string(req.Density),
	})
	if err != nil {
		failAppearance(c, err)
		return
	}
	resp := toGenAppearance(*a)
	c.JSON(http.StatusOK, gensettings.AppearanceResponse{Data: &resp})
}

// appearanceValidationErrors answer 422 with their own code (domain/errs),
// e.g. APPEARANCE_ACCENT_LOW_CONTRAST.
var appearanceValidationErrors = []error{
	settingsDomain.ErrInvalidAppearanceFont,
	settingsDomain.ErrInvalidAppearanceColor,
	settingsDomain.ErrAppearanceAccentLowContrast,
	settingsDomain.ErrInvalidAppearanceCorners,
	settingsDomain.ErrInvalidAppearanceTextSize,
	settingsDomain.ErrInvalidAppearanceDensity,
}

func failAppearance(c *gin.Context, err error) {
	for _, known := range appearanceValidationErrors {
		if errors.Is(err, known) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
	}
	log.Printf("settings: appearance: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
}

func toGenAppearance(a settingsDomain.Appearance) gensettings.Appearance {
	out := gensettings.Appearance{
		Font: gensettings.AppearanceFont(a.Font), AccentColor: string(a.AccentColor), SidebarColor: string(a.SidebarColor),
		Corners: gensettings.AppearanceCorners(a.Corners), TextSize: gensettings.AppearanceTextSize(a.TextSize),
		Density: gensettings.AppearanceDensity(a.Density),
	}
	if !a.UpdatedAt.IsZero() { // zero: never saved, still the defaults
		updatedAt := a.UpdatedAt
		out.UpdatedAt = &updatedAt
	}
	return out
}
