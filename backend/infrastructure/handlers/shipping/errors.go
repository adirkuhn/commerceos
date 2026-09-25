package shipping

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message}})
}

func mapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, shippingDomain.ErrShipmentNotFound), errors.Is(err, integrationsDomain.ErrNoTenantContext),
		errors.Is(err, shippingDomain.ErrAddressNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, shippingDomain.ErrMissingRecipient), errors.Is(err, shippingDomain.ErrInvalidAddressLookupInput):
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, shippingDomain.ErrWebhookSecretNotSet):
		writeError(c, http.StatusConflict, "webhook_secret_not_set", err.Error())
	case errors.Is(err, shippingDomain.ErrDeliveryEstimateUnsupportedCountry):
		writeError(c, http.StatusUnprocessableEntity, "unsupported_country", err.Error())
	default:
		writeError(c, http.StatusBadGateway, "upstream_error", err.Error())
	}
}
