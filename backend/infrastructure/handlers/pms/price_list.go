package pms

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
)

func (h *Handler) ListPriceList(c *gin.Context, params genpms.ListPriceListParams) {
	in := apppms.ListPriceListInput{}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	if params.Search != nil {
		in.Search = *params.Search
	}
	if params.ProductIds != nil {
		in.ProductIDs = make([]string, len(*params.ProductIds))
		for i, id := range *params.ProductIds {
			in.ProductIDs[i] = id.String()
		}
	}
	result, err := h.listPriceList.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genpms.PriceListRow, len(result.Rows))
	for i, r := range result.Rows {
		rows[i] = toGenPriceListRow(r)
	}
	resp := genpms.PriceListResponse{Data: &rows}
	resp.Meta = &struct {
		PricesIncludeTax bool  `json:"prices_include_tax"`
		Total            int64 `json:"total"`
	}{PricesIncludeTax: result.PricesIncludeTax, Total: result.Total}
	c.JSON(http.StatusOK, resp)
}

func toGenPriceListRow(r apppms.PriceListRow) genpms.PriceListRow {
	row := genpms.PriceListRow{
		ProductId: r.ProductID, ProductName: r.ProductName, VariantName: r.VariantName,
		SkuId: r.SKUID, SkuCode: r.SKUCode, UnitType: genpms.SKUUnitType(r.UnitType), CurrencyCode: r.CurrencyCode,
		PriceMinorUnits: r.PriceMinorUnits, SalePriceMinorUnits: r.SalePriceMinorUnits, CostPriceMinorUnits: r.CostPriceMinorUnits,
		AvailableQuantity: r.AvailableQuantity, TaxRateBasisPoints: r.TaxRateBasisPoints, TaxRateConfigured: r.TaxRateConfigured, TargetMarkupBasisPoints: r.TargetMarkupBasisPoints,
	}
	if r.EarliestExpiry != nil {
		row.EarliestExpiry = &openapi_types.Date{Time: *r.EarliestExpiry}
	}
	return row
}
