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
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/domain/locale"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type getTenantSettingsUC interface {
	Execute(ctx context.Context) (*settingsDomain.TenantSettings, error)
}
type updateTenantSettingsUC interface {
	Execute(ctx context.Context, in appsettings.PatchTenantSettingsInput) (*settingsDomain.TenantSettings, error)
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
	*CompanyProfileHandler
	*NotificationPreferencesHandler
	*AppearanceHandler
}

func NewHandler(
	getTenantSettings getTenantSettingsUC,
	updateTenantSettings updateTenantSettingsUC,
	listAPIKeys listAPIKeysUC,
	createAPIKey createAPIKeyUC,
	revokeAPIKey revokeAPIKeyUC,
	companyProfile *CompanyProfileHandler,
	notificationPreferences *NotificationPreferencesHandler,
	appearance *AppearanceHandler,
) *Handler {
	return &Handler{
		getTenantSettings: getTenantSettings, updateTenantSettings: updateTenantSettings,
		listAPIKeys: listAPIKeys, createAPIKey: createAPIKey, revokeAPIKey: revokeAPIKey,
		CompanyProfileHandler: companyProfile, NotificationPreferencesHandler: notificationPreferences,
		AppearanceHandler: appearance,
	}
}

var _ gensettings.ServerInterface = (*Handler)(nil)

// Register wires the settings routes. outbox receives the events settings
// writes raise (e.g. settings.company_profile.updated).
func Register(rg gin.IRouter, db *gorm.DB, outbox appsettings.EventEnqueuer) {
	repo := repositories.NewTenantSettingsRepo(db)
	apiKeyRepo := repositories.NewAPIKeyRepo(db)
	companyRepo := repositories.NewCompanyProfileRepo(db)
	prefsRepo := repositories.NewNotificationPreferencesRepo(db)
	appearanceRepo := repositories.NewAppearanceRepo(db)
	getSettings := appsettings.NewGetTenantSettingsUseCase(db, repo)
	h := NewHandler(
		getSettings,
		appsettings.NewPatchTenantSettingsUseCase(getSettings, appsettings.NewUpdateTenantSettingsUseCase(db, repo, outbox)),
		appapikeys.NewListAPIKeysUseCase(db, apiKeyRepo),
		appapikeys.NewCreateAPIKeyUseCase(db, apiKeyRepo, outbox),
		appapikeys.NewRevokeAPIKeyUseCase(db, apiKeyRepo, outbox),
		NewCompanyProfileHandler(
			appsettings.NewGetCompanyProfileUseCase(db, companyRepo),
			appsettings.NewUpdateCompanyProfileUseCase(db, companyRepo, outbox),
		),
		NewNotificationPreferencesHandler(
			appnotifications.NewGetPreferencesUseCase(db, prefsRepo),
			appnotifications.NewUpdatePreferencesUseCase(db, prefsRepo, outbox),
		),
		NewAppearanceHandler(
			appsettings.NewGetAppearanceUseCase(db, appearanceRepo),
			appsettings.NewUpdateAppearanceUseCase(db, appearanceRepo, outbox),
		),
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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	s, err := h.updateTenantSettings.Execute(c.Request.Context(), appsettings.PatchTenantSettingsInput{
		DefaultCurrency:                req.DefaultCurrency,
		Timezone:                       req.Timezone,
		DateFormat:                     (*string)(req.DateFormat),
		Language:                       (*string)(req.Language),
		PricesIncludeTax:               req.PricesIncludeTax,
		DefaultTaxCountry:              req.DefaultTaxCountry,
		InventorySyncRulesEnabled:      req.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       req.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: req.DefaultTargetMarkupBasisPoints,
		BundleConfidenceThreshold:      float64Ptr(req.BundleConfidenceThreshold),
		RecommendationsMinSupport:      req.RecommendationsMinSupport,
		RecommendationsLookbackDays:    req.RecommendationsLookbackDays,
		PickingMode:                    (*string)(req.PickingMode),
		ExpiryAlertEnabled:             req.ExpiryAlertEnabled,
		ExpiryAlertWindows:             expiryWindowsPtr(req.ExpiryAlertWindows),
		LowStockThreshold:              req.LowStockThreshold,
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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
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
		DateFormat:                     gensettings.DateFormat(s.DateFormat),
		Language:                       gensettings.Language(s.Language.OrDefault()),
		PricesIncludeTax:               s.PricesIncludeTax,
		DefaultTaxCountry:              s.DefaultTaxCountry,
		InventorySyncRulesEnabled:      s.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       s.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: s.DefaultTargetMarkupBasisPoints,
		BundleConfidenceThreshold:      float32(s.BundleConfidenceThreshold),
		RecommendationsMinSupport:      s.RecommendationsMinSupport,
		RecommendationsLookbackDays:    s.RecommendationsLookbackDays,
		PickingMode:                    gensettings.PickingMode(s.PickingMode),
		ExpiryAlertEnabled:             s.ExpiryAlert.Enabled,
		ExpiryAlertWindows:             expiryWindowsOut(s.ExpiryAlert.Windows),
		LowStockThreshold:              s.LowStockThreshold,
		UpdatedAt:                      s.UpdatedAt,
	}
}

func expiryWindowsPtr(w *[]gensettings.ExpiryAlertWindow) *[]int {
	if w == nil {
		return nil
	}
	out := make([]int, len(*w))
	for i, d := range *w {
		out[i] = int(d)
	}
	return &out
}

func float64Ptr(f *float32) *float64 {
	if f == nil {
		return nil
	}
	v := float64(*f)
	return &v
}

func expiryWindowsOut(w []int) []gensettings.ExpiryAlertWindow {
	out := make([]gensettings.ExpiryAlertWindow, len(w))
	for i, d := range w {
		out[i] = gensettings.ExpiryAlertWindow(d)
	}
	return out
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, settingsDomain.ErrInvalidTargetMarkupBasisPoints) || errors.Is(err, settingsDomain.ErrInvalidBundleConfidenceThreshold) || errors.Is(err, settingsDomain.ErrInvalidRecommendationsMinSupport) || errors.Is(err, settingsDomain.ErrInvalidRecommendationsLookbackDays) || errors.Is(err, settingsDomain.ErrInvalidLowStockThreshold) || errors.Is(err, settingsDomain.ErrInvalidPickingMode) || errors.Is(err, settingsDomain.ErrInvalidExpiryAlertWindow) || errors.Is(err, settingsDomain.ErrExpiryAlertWindowsRequired) || errors.Is(err, settingsDomain.ErrInvalidTimezone) || errors.Is(err, settingsDomain.ErrInvalidDateFormat) || errors.Is(err, locale.ErrUnsupported) || errors.Is(err, apikeysDomain.ErrNameRequired) {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return err
	}
	if errors.Is(err, apikeysDomain.ErrNotFound) {
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
		return err
	}
	log.Printf("settings: unhandled error: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	return err
}
