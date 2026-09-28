package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListSalesChannels(c *gin.Context) {
	list, err := h.listSalesChannels.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SalesChannel, len(list))
	for i, ch := range list {
		resp[i] = toGenSalesChannel(ch)
	}
	middleware.OK(c, resp)
}

func (h *Handler) CreateSalesChannel(c *gin.Context) {
	var req genintegrations.CreateSalesChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	tag := ""
	if req.EnrollmentTag != nil {
		tag = *req.EnrollmentTag
	}
	currencyCode := ""
	if req.CurrencyCode != nil {
		currencyCode = *req.CurrencyCode
	}
	channel, err := h.createSalesChannel.Execute(c.Request.Context(), appchannels.CreateSalesChannelInput{
		Name: req.Name, Type: channelsDomain.ChannelType(req.Type), ConnectionID: req.ConnectionId,
		EnrollmentMode: channelsDomain.EnrollmentMode(req.EnrollmentMode), EnrollmentCategoryID: req.EnrollmentCategoryId,
		EnrollmentTag: tag, PriceAdjustmentBasisPoints: req.PriceAdjustmentBasisPoints,
		QuantityBreaks: toDomainQuantityBreaks(req.QuantityBreaks),
		CurrencyCode:   currencyCode, DefaultWarehouseID: req.DefaultWarehouseId,
		StorefrontMode: toDomainStorefrontMode(req.StorefrontMode),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenSalesChannel(*channel)})
}

func (h *Handler) UpdateSalesChannel(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpdateSalesChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	tag := ""
	if req.EnrollmentTag != nil {
		tag = *req.EnrollmentTag
	}
	channel, err := h.updateSalesChannel.Execute(c.Request.Context(), appchannels.UpdateSalesChannelInput{
		ID: id, Name: req.Name, ConnectionID: req.ConnectionId,
		EnrollmentMode: channelsDomain.EnrollmentMode(req.EnrollmentMode), EnrollmentCategoryID: req.EnrollmentCategoryId,
		EnrollmentTag: tag, PriceAdjustmentBasisPoints: req.PriceAdjustmentBasisPoints, Active: req.Active,
		QuantityBreaks:     toDomainQuantityBreaks(req.QuantityBreaks),
		DefaultWarehouseID: req.DefaultWarehouseId,
		StorefrontMode:     toDomainStorefrontMode(req.StorefrontMode),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenSalesChannel(*channel))
}

func (h *Handler) SetSalesChannelPaymentMethod(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.SetSalesChannelPaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	channel, err := h.setSalesChannelPaymentMethod.Execute(c.Request.Context(), id, req.PaymentConnectionId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenSalesChannel(*channel))
}

func (h *Handler) DeleteSalesChannel(c *gin.Context, id genintegrations.Id) {
	err := h.deleteSalesChannel.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListSalesChannelProducts(c *gin.Context, id genintegrations.Id, params genintegrations.ListSalesChannelProductsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	rows, total, err := h.listSalesChannelProducts.Execute(c.Request.Context(), appchannels.ListProductsWithChannelEnrollmentInput{
		ChannelID: id, Search: params.Search, Limit: perPage, Offset: (page - 1) * perPage,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SalesChannelProductEnrollment, len(rows))
	for i, r := range rows {
		resp[i] = genintegrations.SalesChannelProductEnrollment{Enrolled: r.Enrolled, Product: toGenSalesChannelProduct(r.Product)}
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, genintegrations.SalesChannelProductListResponse{Data: &resp, Total: &totalInt})
}

func (h *Handler) SetSalesChannelProductOverride(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	var req genintegrations.SetSalesChannelProductOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	_, err := h.setProductChannelOverride.Execute(c.Request.Context(), appchannels.SetProductChannelOverrideInput{
		ChannelID: id, ProductID: productId, Included: req.Included,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenSalesChannel(c channelsDomain.SalesChannel) genintegrations.SalesChannel {
	resp := genintegrations.SalesChannel{
		Id: c.ID, Name: c.Name, Type: genintegrations.ChannelType(c.Type),
		ConnectionId: c.ConnectionID, EnrollmentMode: genintegrations.EnrollmentMode(c.EnrollmentMode),
		EnrollmentCategoryId: c.EnrollmentCategoryID,
		PaymentConnectionId:  c.PaymentConnectionID,
		// PriceAdjustmentBasisPoints was missing here entirely -- a real
		// bug (the field the admin form needs to re-show a channel's own
		// discount when reopening it for edit, and the sales-channels
		// list's "20% off" badge, both silently always saw nil) fixed
		// alongside adding QuantityBreaks below.
		PriceAdjustmentBasisPoints: c.PriceAdjustmentBasisPoints,
		QuantityBreaks:             toGenQuantityBreaks(c.QuantityBreaks),
		DefaultWarehouseId:         c.DefaultWarehouseID,
		StorefrontMode:             genintegrations.StorefrontMode(c.StorefrontMode),
		Active:                     c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.EnrollmentTag != "" {
		resp.EnrollmentTag = &c.EnrollmentTag
	}
	if c.CurrencyCode != "" {
		resp.CurrencyCode = &c.CurrencyCode
	}
	return resp
}

// toDomainStorefrontMode maps an omitted mode to "" -- the use cases read
// that as headless on create and "keep current" on update.
func toDomainStorefrontMode(m *genintegrations.StorefrontMode) channelsDomain.StorefrontMode {
	if m == nil {
		return ""
	}
	return channelsDomain.StorefrontMode(*m)
}

func toDomainQuantityBreaks(breaks *[]genintegrations.QuantityBreak) []channelsDomain.QuantityBreak {
	if breaks == nil {
		return nil
	}
	out := make([]channelsDomain.QuantityBreak, len(*breaks))
	for i, b := range *breaks {
		out[i] = channelsDomain.QuantityBreak{MinQuantity: int64(b.MinQuantity), DiscountBasisPoints: b.DiscountBasisPoints}
	}
	return out
}

func toGenQuantityBreaks(breaks []channelsDomain.QuantityBreak) *[]genintegrations.QuantityBreak {
	out := make([]genintegrations.QuantityBreak, len(breaks))
	for i, b := range breaks {
		out[i] = genintegrations.QuantityBreak{MinQuantity: int(b.MinQuantity), DiscountBasisPoints: b.DiscountBasisPoints}
	}
	return &out
}

func toGenSalesChannelProduct(p pmsDomain.Product) genintegrations.SalesChannelProduct {
	resp := genintegrations.SalesChannelProduct{Id: p.ID, Name: p.Name, Slug: p.Slug, CategoryId: p.CategoryID}
	if len(p.Tags) > 0 {
		tags := p.Tags
		resp.Tags = &tags
	}
	return resp
}
