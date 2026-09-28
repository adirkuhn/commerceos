package integrations

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, integrationsDomain.ErrConnectionNotFound),
		errors.Is(err, pmsDomain.ErrProductNotFound),
		errors.Is(err, pmsDomain.ErrSKUNotFound),
		errors.Is(err, pmsDomain.ErrCategoryNotFound),
		errors.Is(err, channelsDomain.ErrListingNotFound),
		errors.Is(err, channelsDomain.ErrSalesChannelNotFound),
		errors.Is(err, channelsDomain.ErrExternalCategoryLinkNotFound),
		errors.Is(err, channelsDomain.ErrSyncLogNotFound),
		errors.Is(err, webhooksDomain.ErrFallbackEventNotFound),
		errors.Is(err, webhooksDomain.ErrWebhookEventNotFound),
		errors.Is(err, integrationsDomain.ErrWooProductNotFoundBySKU):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, webhooksDomain.ErrWebhookEventNotDispatchable):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrSyncLogHasNoListing):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrStoreURLInvalid):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrProviderNotSupported),
		errors.Is(err, integrationsDomain.ErrAPIKeyRequired),
		errors.Is(err, integrationsDomain.ErrBaseURLRequired),
		errors.Is(err, integrationsDomain.ErrInvalidBolDeliveryCode),
		errors.Is(err, integrationsDomain.ErrWhatsAppTestPhoneRequired),
		errors.Is(err, integrationsDomain.ErrNotWhatsAppConnection),
		errors.Is(err, appintegrations.ErrAIConnectionRequiredForWhatsAppParsing):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrCannotPushVariableProduct),
		errors.Is(err, channelsDomain.ErrInvalidSyncEntityType),
		errors.Is(err, channelsDomain.ErrInvalidSyncDirection),
		errors.Is(err, channelsDomain.ErrInvalidSyncInterval),
		errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity),
		errors.Is(err, channelsDomain.ErrDefaultWarehouseAndCurrencyRequired),
		errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired),
		errors.Is(err, channelsDomain.ErrInvalidChannelType),
		errors.Is(err, channelsDomain.ErrInvalidEnrollmentMode),
		errors.Is(err, channelsDomain.ErrInvalidStorefrontMode),
		errors.Is(err, channelsDomain.ErrThemedStorefrontRequiresInternal),
		errors.Is(err, channelsDomain.ErrConnectionRequired),
		errors.Is(err, channelsDomain.ErrEnrollmentCategoryRequired),
		errors.Is(err, channelsDomain.ErrEnrollmentTagRequired),
		errors.Is(err, channelsDomain.ErrInvalidPaymentConnection),
		errors.Is(err, channelsDomain.ErrChannelTypeConnectionMismatch),
		errors.Is(err, channelsDomain.ErrProductAlreadyLinkedToConnection),
		errors.Is(err, channelsDomain.ErrInvalidPriceAdjustment),
		errors.Is(err, channelsDomain.ErrInvalidQuantityBreak):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrStateInvalid):
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, integrationsDomain.ErrConnectionTestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrWooRequestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrNoTenantContext):
		middleware.Fail(c, http.StatusBadRequest, "NO_TENANT", err.Error())
	case errors.Is(err, channelsDomain.ErrImportJobAlreadyActive):
		middleware.Fail(c, http.StatusConflict, "IMPORT_ALREADY_RUNNING", err.Error())
	case errors.Is(err, channelsDomain.ErrBolMissingEAN),
		errors.Is(err, channelsDomain.ErrCannotPushBulkPoolSKU):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrBolCommissionRequestFailed),
		errors.Is(err, integrationsDomain.ErrBolEconomicOperatorsRequestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	default:
		log.Printf("integrations: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
