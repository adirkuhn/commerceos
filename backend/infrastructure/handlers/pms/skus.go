package pms

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// genDateToTimePtr/timePtrToGenDate convert between openapi_types.Date
// (date-only, no time-of-day) and *time.Time -- same helpers as
// infrastructure/handlers/inventory's, duplicated rather than shared
// since these two handler packages don't otherwise depend on each other.
func genDateToTimePtr(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

// genSKUUnitTypeOrEmpty maps an optional request unit_type to the domain
// type -- nil becomes "" (empty), which CreateSKUUseCase/UpdateSKUUseCase
// normalize to SKUUnitTypeUnit, matching the OpenAPI field's documented
// "defaults to unit" behavior.
func genSKUUnitTypeOrEmpty(t *genpms.SKUUnitType) pmsDomain.SKUUnitType {
	if t == nil {
		return ""
	}
	return pmsDomain.SKUUnitType(*t)
}

func genSKUPricingModeOrEmpty(m *genpms.SKUPricingMode) pmsDomain.SKUPricingMode {
	if m == nil {
		return ""
	}
	return pmsDomain.SKUPricingMode(*m)
}

func timePtrToGenDate(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

// ── SKU handlers ──────────────────────────────────────────────────────────────

func (h *Handler) CreateSKU(c *gin.Context, id genpms.Id) {
	var req genpms.CreateSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	barcode := ""
	if req.Barcode != nil {
		barcode = *req.Barcode
	}
	sku, err := h.createSKU.Execute(c.Request.Context(), apppms.CreateSKUInput{
		VariantID:              id,
		SKUCode:                req.SkuCode,
		Barcode:                barcode,
		PriceMinorUnits:        req.PriceMinorUnits,
		SalePriceMinorUnits:    req.SalePriceMinorUnits,
		SaleEndDate:            genDateToTimePtr(req.SaleEndDate),
		CurrencyCode:           req.CurrencyCode,
		WeightGrams:            req.WeightGrams,
		LengthMM:               req.LengthMm,
		WidthMM:                req.WidthMm,
		HeightMM:               req.HeightMm,
		IsVirtual:              req.IsVirtual != nil && *req.IsVirtual,
		TracksExpiry:           req.TracksExpiry != nil && *req.TracksExpiry,
		CaseOfSKUID:            req.CaseOfSkuId,
		UnitsPerCase:           req.UnitsPerCase,
		UnitType:               genSKUUnitTypeOrEmpty(req.UnitType),
		PortionedFromSKUID:     req.PortionedFromSkuId,
		ScaleCode:              req.ScaleCode,
		PricingMode:            genSKUPricingModeOrEmpty(req.PricingMode),
		PricePerKiloMinorUnits: req.PricePerKiloMinorUnits,
		TaxClassID:             req.TaxClassId,
		Status:                 pmsDomain.SKUStatus(req.Status),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusCreated, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) UpdateSKU(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	barcode := ""
	if req.Barcode != nil {
		barcode = *req.Barcode
	}
	sku, err := h.updateSKU.Execute(c.Request.Context(), apppms.UpdateSKUInput{
		ID:                     id,
		SKUCode:                req.SkuCode,
		Barcode:                barcode,
		PriceMinorUnits:        req.PriceMinorUnits,
		SalePriceMinorUnits:    req.SalePriceMinorUnits,
		SaleEndDate:            genDateToTimePtr(req.SaleEndDate),
		CurrencyCode:           req.CurrencyCode,
		WeightGrams:            req.WeightGrams,
		LengthMM:               req.LengthMm,
		WidthMM:                req.WidthMm,
		HeightMM:               req.HeightMm,
		IsVirtual:              req.IsVirtual != nil && *req.IsVirtual,
		TracksExpiry:           req.TracksExpiry != nil && *req.TracksExpiry,
		CaseOfSKUID:            req.CaseOfSkuId,
		UnitsPerCase:           req.UnitsPerCase,
		UnitType:               genSKUUnitTypeOrEmpty(req.UnitType),
		PortionedFromSKUID:     req.PortionedFromSkuId,
		ScaleCode:              req.ScaleCode,
		PricingMode:            genSKUPricingModeOrEmpty(req.PricingMode),
		PricePerKiloMinorUnits: req.PricePerKiloMinorUnits,
		TaxClassID:             req.TaxClassId,
		Status:                 pmsDomain.SKUStatus(req.Status),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusOK, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) DeleteSKU(c *gin.Context, id genpms.Id) {
	err := h.deleteSKU.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func toGenSKU(s pmsDomain.SKU) genpms.SKU {
	var barcode *string
	if s.Barcode != "" {
		barcode = &s.Barcode
	}
	unitType := genpms.SKUUnitType(s.UnitType)
	pricingMode := genpms.SKUPricingMode(s.PricingMode)
	return genpms.SKU{
		Id:                     s.ID,
		TenantId:               s.TenantID,
		VariantId:              s.VariantID,
		SkuCode:                s.SKUCode,
		Barcode:                barcode,
		PriceMinorUnits:        s.PriceMinorUnits,
		SalePriceMinorUnits:    s.SalePriceMinorUnits,
		SaleEndDate:            timePtrToGenDate(s.SaleEndDate),
		CurrencyCode:           s.CurrencyCode,
		WeightGrams:            s.WeightGrams,
		LengthMm:               s.LengthMM,
		WidthMm:                s.WidthMM,
		HeightMm:               s.HeightMM,
		IsVirtual:              s.IsVirtual,
		TracksExpiry:           s.TracksExpiry,
		CaseOfSkuId:            s.CaseOfSKUID,
		UnitsPerCase:           s.UnitsPerCase,
		UnitType:               &unitType,
		PortionedFromSkuId:     s.PortionedFromSKUID,
		ScaleCode:              s.ScaleCode,
		PricingMode:            &pricingMode,
		PricePerKiloMinorUnits: s.PricePerKiloMinorUnits,
		TaxClassId:             s.TaxClassID,
		WooTaxClassSlug:        s.WooTaxClassSlug,
		WooTaxStatus:           s.WooTaxStatus,
		Status:                 genpms.SKUStatus(s.Status),
		CostPriceMinorUnits:    s.CostPriceMinorUnits,
		CostPriceSource:        costPriceSourcePtr(s.CostPriceSource),
		CreatedAt:              s.CreatedAt,
		UpdatedAt:              s.UpdatedAt,
	}
}

// costPriceSourcePtr defaults an unset (zero-value) CostPriceSource to
// "computed" -- matches the DB column's own default, for SKU rows read
// before this field existed in the domain struct at all (shouldn't happen
// post-migration, but avoids emitting an empty string over the wire).
func costPriceSourcePtr(s pmsDomain.SKUCostPriceSource) *genpms.SKUCostPriceSource {
	if s == "" {
		s = pmsDomain.SKUCostPriceSourceComputed
	}
	v := genpms.SKUCostPriceSource(s)
	return &v
}
