package settings

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gensettings "github.com/adirkuhn/commerceos/api/settings"
	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
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
type listAPIKeysUC interface {
	Execute(ctx context.Context) ([]apikeysDomain.APIKey, error)
}
type createAPIKeyUC interface {
	Execute(ctx context.Context, name, description string, salesChannelID *string) (*appapikeys.CreateAPIKeyResult, error)
}
type revokeAPIKeyUC interface {
	Execute(ctx context.Context, id string) error
}

type Handler struct {
	getTenantSettings    getTenantSettingsUC
	updateTenantSettings updateTenantSettingsUC
	listAPIKeys          listAPIKeysUC
	createAPIKey         createAPIKeyUC
	revokeAPIKey         revokeAPIKeyUC
}

func NewHandler(
	getTenantSettings getTenantSettingsUC,
	updateTenantSettings updateTenantSettingsUC,
	listAPIKeys listAPIKeysUC,
	createAPIKey createAPIKeyUC,
	revokeAPIKey revokeAPIKeyUC,
) *Handler {
	return &Handler{
		getTenantSettings: getTenantSettings, updateTenantSettings: updateTenantSettings,
		listAPIKeys: listAPIKeys, createAPIKey: createAPIKey, revokeAPIKey: revokeAPIKey,
	}
}

var _ gensettings.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewTenantSettingsRepo(db)
	apiKeyRepo := repositories.NewAPIKeyRepo(db)
	h := NewHandler(
		appsettings.NewGetTenantSettingsUseCase(db, repo),
		appsettings.NewUpdateTenantSettingsUseCase(db, repo),
		appapikeys.NewListAPIKeysUseCase(db, apiKeyRepo),
		appapikeys.NewCreateAPIKeyUseCase(db, apiKeyRepo),
		appapikeys.NewRevokeAPIKeyUseCase(db, apiKeyRepo),
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
		BundleConfidenceThreshold:      float64PtrOr(req.BundleConfidenceThreshold),
		RecommendationsMinSupport:      intPtrOr(req.RecommendationsMinSupport),
		RecommendationsLookbackDays:    intPtrOr(req.RecommendationsLookbackDays),
		PickingMode:                    pickingModeOr(req.PickingMode),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenTenantSettings(*s)
	c.JSON(http.StatusOK, gensettings.TenantSettingsResponse{Data: &resp})
}

func (h *Handler) ListApiKeys(c *gin.Context) {
	keys, err := h.listAPIKeys.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]gensettings.ApiKey, len(keys))
	for i, k := range keys {
		items[i] = toGenAPIKey(k)
	}
	middleware.OK(c, struct {
		Items []gensettings.ApiKey `json:"items"`
	}{Items: items})
}

func (h *Handler) CreateApiKey(c *gin.Context) {
	var req gensettings.CreateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	result, err := h.createAPIKey.Execute(c.Request.Context(), req.Name, description, req.SalesChannelId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gensettings.ApiKeyCreated{
		Id: result.ID, Name: result.Name, Description: result.Description, SalesChannelId: result.SalesChannelID,
		CreatedAt: result.CreatedAt, Key: result.PlaintextKey,
	})
}

func (h *Handler) RevokeApiKey(c *gin.Context, id string) {
	err := h.revokeAPIKey.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenAPIKey(k apikeysDomain.APIKey) gensettings.ApiKey {
	return gensettings.ApiKey{
		Id: k.ID, Name: k.Name, Description: k.Description, CreatedAt: k.CreatedAt, RevokedAt: k.RevokedAt,
		SalesChannelId: k.SalesChannelID,
	}
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
		BundleConfidenceThreshold:      float32(s.BundleConfidenceThreshold),
		RecommendationsMinSupport:      s.RecommendationsMinSupport,
		RecommendationsLookbackDays:    s.RecommendationsLookbackDays,
		PickingMode:                    gensettings.PickingMode(s.PickingMode),
		UpdatedAt:                      s.UpdatedAt,
	}
}

func pickingModeOr(m *gensettings.PickingMode) string {
	if m == nil {
		return ""
	}
	return string(*m)
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

func float64PtrOr(f *float32) float64 {
	if f == nil {
		return 0
	}
	return float64(*f)
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, settingsDomain.ErrInvalidTargetMarkupBasisPoints) || errors.Is(err, settingsDomain.ErrInvalidBundleConfidenceThreshold) || errors.Is(err, settingsDomain.ErrInvalidRecommendationsMinSupport) || errors.Is(err, settingsDomain.ErrInvalidRecommendationsLookbackDays) || errors.Is(err, settingsDomain.ErrInvalidPickingMode) || errors.Is(err, apikeysDomain.ErrNameRequired) {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return err
	}
	if errors.Is(err, apikeysDomain.ErrNotFound) {
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return err
	}
	log.Printf("settings: unhandled error: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	return err
}
