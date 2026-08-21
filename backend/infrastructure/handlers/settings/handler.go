package settings

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gensettings "github.com/adirkuhn/commerceos/api/settings"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type getTenantSettingsUC interface {
	Execute(ctx context.Context) (*settingsDomain.TenantSettings, error)
}
type updateTenantSettingsUC interface {
	Execute(ctx context.Context, in appsettings.UpdateTenantSettingsInput) (*settingsDomain.TenantSettings, error)
}

type Handler struct {
	getTenantSettings    getTenantSettingsUC
	updateTenantSettings updateTenantSettingsUC
}

func NewHandler(getTenantSettings getTenantSettingsUC, updateTenantSettings updateTenantSettingsUC) *Handler {
	return &Handler{getTenantSettings: getTenantSettings, updateTenantSettings: updateTenantSettings}
}

var _ gensettings.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewTenantSettingsRepo(db)
	h := NewHandler(
		appsettings.NewGetTenantSettingsUseCase(db, repo),
		appsettings.NewUpdateTenantSettingsUseCase(db, repo),
	)
	gensettings.RegisterHandlers(rg, h)
}

func (h *Handler) GetTenantSettings(c *gin.Context) {
	s, err := h.getTenantSettings.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenTenantSettings(*s)
	c.JSON(http.StatusOK, gensettings.TenantSettingsResponse{Data: &resp})
}

func (h *Handler) UpdateTenantSettings(c *gin.Context) {
	var req gensettings.UpdateTenantSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	s, err := h.updateTenantSettings.Execute(c.Request.Context(), appsettings.UpdateTenantSettingsInput{
		DefaultCurrency:                strPtrOr(req.DefaultCurrency),
		Timezone:                       strPtrOr(req.Timezone),
		DateFormat:                     strPtrOr(req.DateFormat),
		Language:                       strPtrOr(req.Language),
		PricesIncludeTax:               req.PricesIncludeTax != nil && *req.PricesIncludeTax,
		DefaultTaxCountry:              strPtrOr(req.DefaultTaxCountry),
		InventorySyncRulesEnabled:      req.InventorySyncRulesEnabled != nil && *req.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       intPtrOr(req.BatchExpiryThresholdDays),
		DefaultTargetMarkupBasisPoints: intPtrOr(req.DefaultTargetMarkupBasisPoints),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenTenantSettings(*s)
	c.JSON(http.StatusOK, gensettings.TenantSettingsResponse{Data: &resp})
}

func toGenTenantSettings(s settingsDomain.TenantSettings) gensettings.TenantSettings {
	return gensettings.TenantSettings{
		TenantId:                       s.TenantID,
		DefaultCurrency:                s.DefaultCurrency,
		Timezone:                       s.Timezone,
		DateFormat:                     s.DateFormat,
		Language:                       s.Language,
		PricesIncludeTax:               s.PricesIncludeTax,
		DefaultTaxCountry:              s.DefaultTaxCountry,
		InventorySyncRulesEnabled:      s.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       s.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: s.DefaultTargetMarkupBasisPoints,
		UpdatedAt:                      s.UpdatedAt,
	}
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtrOr(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, settingsDomain.ErrInvalidTargetMarkupBasisPoints) {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return err
	}
	log.Printf("settings: unhandled error: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	return err
}
