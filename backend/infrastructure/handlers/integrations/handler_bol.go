package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) PushProductToBol(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.PushProductToWooRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.ProductId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "product_id is required")
		return
	}
	listings, err := h.pushProductToBol.Execute(c.Request.Context(), appchannels.PushProductToBolInput{
		ProductID: req.ProductId, ConnectionID: id,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListings(listings))
}

func (h *Handler) GetBolProductContent(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	content, err := h.getBolProductContent.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	if content == nil {
		middleware.OK(c, nil)
		return
	}
	middleware.OK(c, toGenBolProductContent(*content))
}

func (h *Handler) UpsertBolProductContent(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	var req genintegrations.UpsertBolProductContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	content, err := h.upsertBolProductContent.Execute(c.Request.Context(), appchannels.UpsertBolProductContentInput{
		ProductID: productId, ConnectionID: id,
		ManufacturerContact: req.ManufacturerContact, ManufacturerAddress: req.ManufacturerAddress, ManufacturerName: req.ManufacturerName,
		ShelfLifeClosed: req.ShelfLifeClosed, ShelfLifeAfterOpening: req.ShelfLifeAfterOpening, Ingredients: req.Ingredients,
		ContentAmount: req.ContentAmount, ContentUnit: req.ContentUnit, CountryOfOrigin: req.CountryOfOrigin,
		Substance: req.Substance, InstructionLanguage: req.InstructionLanguage, PackagingLanguage: req.PackagingLanguage,
		ItemsPerPackage: intToInt32Ptr(req.ItemsPerPackage), PackageInsertURL: req.PackageInsertUrl, IsGift: req.IsGift,
		DermatologicallyTested: req.DermatologicallyTested, HazardStatements: req.HazardStatements, SkinType: req.SkinType,
		Hypoallergenic: req.Hypoallergenic, ContainsZincOxide: req.ContainsZincOxide,
		ContentPerDispenserAmount: req.ContentPerDispenserAmount, ContentPerDispenserUnit: req.ContentPerDispenserUnit,
		Brand: req.Brand, FragranceFree: req.FragranceFree, PHNeutral: req.PhNeutral, CoolingEffect: req.CoolingEffect,
		PackageWidthMM: intToInt32Ptr(req.PackageWidthMm), PackageHeightMM: intToInt32Ptr(req.PackageHeightMm),
		PackageLengthMM: intToInt32Ptr(req.PackageLengthMm), PackageWeightG: intToInt32Ptr(req.PackageWeightG),
		PrecautionStatements: req.PrecautionStatements, Precautions: req.Precautions, Warnings: req.Warnings, VideoURL: req.VideoUrl,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenBolProductContent(*content))
}

func (h *Handler) GetBolProductOverride(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	override, err := h.getBolProductOverride.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	if override == nil {
		middleware.OK(c, nil)
		return
	}
	middleware.OK(c, toGenBolOverride(*override))
}

func (h *Handler) UpsertBolProductOverride(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	var req genintegrations.UpsertBolProductOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	condition, fulfilment := "", ""
	if req.ConditionCategory != nil {
		condition = string(*req.ConditionCategory)
	}
	if req.FulfilmentMethod != nil {
		fulfilment = string(*req.FulfilmentMethod)
	}
	var basisPoints *int32
	if req.CommissionOverrideBasisPoints != nil {
		bp := int32(*req.CommissionOverrideBasisPoints)
		basisPoints = &bp
	}
	override, err := h.upsertBolProductOverride.Execute(c.Request.Context(), appchannels.UpsertBolProductOverrideInput{
		ProductID: productId, ConnectionID: id,
		EANOverride: req.EanOverride, GPCCode: req.GpcCode,
		ConditionCategory:             condition,
		FulfilmentMethod:              fulfilment,
		CommissionOverrideBasisPoints: basisPoints,
		ReferenceOverride:             req.ReferenceOverride,
		CustomPriceMinorUnits:         req.CustomPriceMinorUnits,
		NameOverride:                  req.NameOverride,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenBolOverride(*override))
}

func (h *Handler) GetBolCommission(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId, skuId string) {
	result, err := h.getBolCommission.Execute(c.Request.Context(), appchannels.GetBolCommissionInput{
		ProductID: productId, ConnectionID: id, SKUID: skuId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genintegrations.BolCommission{
		Ean: result.EAN, FixedAmount: result.FixedAmount, Percentage: result.Percentage, Overridden: result.Overridden,
	})
}

func (h *Handler) ListBolEconomicOperators(c *gin.Context, id genintegrations.Id) {
	operators, err := h.listBolEconomicOperators.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.BolEconomicOperator, len(operators))
	for i, o := range operators {
		resp[i] = genintegrations.BolEconomicOperator{
			Id: o.ID, Name: o.Name, Status: genintegrations.BolEconomicOperatorStatus(o.Status),
		}
	}
	middleware.OK(c, resp)
}

func int32ToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func intToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := int32(*v)
	return &i
}

func toGenBolProductContent(c channelsDomain.BolProductContent) genintegrations.BolProductContent {
	return genintegrations.BolProductContent{
		ProductId: c.ProductID, ConnectionId: c.ConnectionID,
		ManufacturerContact: c.ManufacturerContact, ManufacturerAddress: c.ManufacturerAddress, ManufacturerName: c.ManufacturerName,
		ShelfLifeClosed: c.ShelfLifeClosed, ShelfLifeAfterOpening: c.ShelfLifeAfterOpening, Ingredients: c.Ingredients,
		ContentAmount: c.ContentAmount, ContentUnit: c.ContentUnit, CountryOfOrigin: c.CountryOfOrigin,
		Substance: c.Substance, InstructionLanguage: c.InstructionLanguage, PackagingLanguage: c.PackagingLanguage,
		ItemsPerPackage: int32ToIntPtr(c.ItemsPerPackage), PackageInsertUrl: c.PackageInsertURL, IsGift: c.IsGift,
		DermatologicallyTested: c.DermatologicallyTested, HazardStatements: c.HazardStatements, SkinType: c.SkinType,
		Hypoallergenic: c.Hypoallergenic, ContainsZincOxide: c.ContainsZincOxide,
		ContentPerDispenserAmount: c.ContentPerDispenserAmount, ContentPerDispenserUnit: c.ContentPerDispenserUnit,
		Brand: c.Brand, FragranceFree: c.FragranceFree, PhNeutral: c.PHNeutral, CoolingEffect: c.CoolingEffect,
		PackageWidthMm: int32ToIntPtr(c.PackageWidthMM), PackageHeightMm: int32ToIntPtr(c.PackageHeightMM),
		PackageLengthMm: int32ToIntPtr(c.PackageLengthMM), PackageWeightG: int32ToIntPtr(c.PackageWeightG),
		PrecautionStatements: c.PrecautionStatements, Precautions: c.Precautions, Warnings: c.Warnings, VideoUrl: c.VideoURL,
	}
}

func toGenBolOverride(o channelsDomain.BolProductOverride) genintegrations.BolProductOverride {
	resp := genintegrations.BolProductOverride{
		ProductId: o.ProductID, ConnectionId: o.ConnectionID,
		EanOverride: o.EANOverride, GpcCode: o.GPCCode,
		ConditionCategory:     genintegrations.BolProductOverrideConditionCategory(o.ConditionCategory),
		FulfilmentMethod:      genintegrations.BolProductOverrideFulfilmentMethod(o.FulfilmentMethod),
		ReferenceOverride:     o.ReferenceOverride,
		CustomPriceMinorUnits: o.CustomPriceMinorUnits,
		NameOverride:          o.NameOverride,
	}
	if o.CommissionOverrideBasisPoints != nil {
		bp := int(*o.CommissionOverrideBasisPoints)
		resp.CommissionOverrideBasisPoints = &bp
	}
	return resp
}
