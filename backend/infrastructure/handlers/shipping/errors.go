package shipping

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message}})
}

func mapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, shippingDomain.ErrShipmentNotFound), errors.Is(err, integrationsDomain.ErrNoTenantContext),
		errors.Is(err, shippingDomain.ErrAddressNotFound):
		middleware.FailErr(c, http.StatusNotFound, "not_found", err)
	case errors.Is(err, shippingDomain.ErrShippingPreferencesForbidden):
		middleware.FailErr(c, http.StatusForbidden, "forbidden", err)
	case errors.Is(err, integrationsDomain.ErrConnectionNotFound):
		middleware.FailErr(c, http.StatusNotFound, "not_found", err)
	case errors.Is(err, integrationsDomain.ErrPostNLAccountInvalid), errors.Is(err, integrationsDomain.ErrCarrierSenderAddressMissing),
		errors.Is(err, integrationsDomain.ErrCarrierOperationNotSupported), errors.Is(err, integrationsDomain.ErrProviderNotSupported):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "unprocessable", err)
	case errors.Is(err, shippingDomain.ErrNotAShippingConnection):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "not_a_shipping_connection", err)
	case errors.Is(err, shippingDomain.ErrMissingRecipient), errors.Is(err, shippingDomain.ErrInvalidAddressLookupInput):
		middleware.FailErr(c, http.StatusBadRequest, "invalid_request", err)
	case errors.Is(err, shippingDomain.ErrWebhookSecretNotSet):
		middleware.FailErr(c, http.StatusConflict, "webhook_secret_not_set", err)
	case errors.Is(err, shippingDomain.ErrDeliveryEstimateUnsupportedCountry):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "unsupported_country", err)
	default:
		middleware.FailErr(c, http.StatusBadGateway, "upstream_error", err)
	}
}
