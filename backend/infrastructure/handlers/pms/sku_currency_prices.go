package pms

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListSKUCurrencyPrices(c *gin.Context, id genpms.Id) {
	prices, err := h.listSKUCurrencyPrices.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.SKUCurrencyPrice, len(prices))
	for i, p := range prices {
		resp[i] = toGenSKUCurrencyPrice(p)
	}
	c.JSON(http.StatusOK, genpms.SKUCurrencyPriceListResponse{Data: &resp})
}

func (h *Handler) UpsertSKUCurrencyPrice(c *gin.Context, id genpms.Id) {
	var req genpms.UpsertSKUCurrencyPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	price, err := h.upsertSKUCurrencyPrice.Execute(c.Request.Context(), apppms.UpsertSKUCurrencyPriceInput{
		SKUID: id, CurrencyCode: req.CurrencyCode, PriceMinorUnits: req.PriceMinorUnits,
		SalePriceMinorUnits: req.SalePriceMinorUnits, SaleEndDate: genDateToTimePtr(req.SaleEndDate),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKUCurrencyPrice(*price)
	c.JSON(http.StatusOK, genpms.SKUCurrencyPriceResponse{Data: &resp})
}

func (h *Handler) DeleteSKUCurrencyPrice(c *gin.Context, id genpms.Id, currencyCode string) {
	err := h.deleteSKUCurrencyPrice.Execute(c.Request.Context(), id, currencyCode)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func toGenSKUCurrencyPrice(p pmsDomain.SKUCurrencyPrice) genpms.SKUCurrencyPrice {
	return genpms.SKUCurrencyPrice{
		SkuId: p.SKUID, CurrencyCode: p.CurrencyCode, PriceMinorUnits: p.PriceMinorUnits,
		SalePriceMinorUnits: p.SalePriceMinorUnits, SaleEndDate: timePtrToGenDate(p.SaleEndDate),
		UpdatedAt: p.UpdatedAt,
	}
}
