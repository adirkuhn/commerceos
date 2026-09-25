package shipping

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func (h *Handler) LookupAddress(c *gin.Context, params genshipping.LookupAddressParams) {
	result, err := h.lookupAddress.Execute(c.Request.Context(), params.PostalCode, params.Number)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"street": result.Street, "city": result.City}, "error": nil})
}

func (h *Handler) GetDeliveryPriceEstimate(c *gin.Context, params genshipping.GetDeliveryPriceEstimateParams) {
	options, err := h.getDeliveryEstimate.Execute(c.Request.Context(), appshipping.GetDeliveryPriceEstimateInput{
		ConnectionID: params.ConnectionId.String(), CC: params.Cc, PostalCode: params.PostalCode,
		Number: params.Number, Carrier: shippingDomain.Carrier(params.Carrier),
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	out := make([]genshipping.DeliveryPriceOption, len(options))
	for i := range options {
		out[i] = genshipping.DeliveryPriceOption{
			PriceComment: &options[i].PriceComment, AmountMinorUnits: &options[i].AmountMinorUnits, Currency: &options[i].Currency,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "error": nil})
}
