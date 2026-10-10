package storefront

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (h *Handler) GetCartRecommendations(c *gin.Context) {
	resolved, ok := h.authenticate(c)
	if !ok {
		return
	}

	var req genstorefront.CartRecommendationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}

	lines := make([]appreports.CartLine, len(req.Items))
	for i, item := range req.Items {
		lines[i] = appreports.CartLine{SKUCode: item.SkuCode, Quantity: item.Quantity}
	}

	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	report, err := h.getCartRecommendations.Execute(ctx, lines)
	if err := mapError(c, err); err != nil {
		return
	}

	recs := make([]genstorefront.RecommendedProduct, len(report.Recommendations))
	for i, r := range report.Recommendations {
		recs[i] = genstorefront.RecommendedProduct{
			SkuCode:           r.SKUCode,
			ProductName:       r.ProductName,
			Confidence:        r.Confidence,
			IsBundleCandidate: r.IsBundleCandidate,
		}
	}
	middleware.OK(c, genstorefront.CartRecommendationsReport{Recommendations: recs, TotalOrders: report.TotalOrders})
}

func (h *Handler) CreateCart(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	cart, err := h.createCart.Execute(ctx, *resolved.SalesChannelID)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, genstorefront.CartResponse{Data: toGenCart(appstorefront.CartResult{Cart: cart})})
}

func (h *Handler) GetCart(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	cart, err := h.getCart.Execute(ctx, token)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCart(*cart))
}

func (h *Handler) UpdateCartItems(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.UpdateCartItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	lines := make([]appstorefront.UpdateCartItemLineInput, len(req.Items))
	for i, item := range req.Items {
		lines[i] = appstorefront.UpdateCartItemLineInput{SKUCode: item.SkuCode, Quantity: int64(item.Quantity)}
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	cart, err := h.updateCartItems.Execute(ctx, token, lines)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCart(*cart))
}

func (h *Handler) ApplyCoupon(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.ApplyCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.applyCoupon.Execute(ctx, token, strPtrOr(req.Code))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCart(*result))
}

func (h *Handler) GetShippingOptions(c *gin.Context, _ genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.CheckoutAddress
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	options := h.getShippingOptions.Execute(ctx, toAddressInput(req))
	genOptions := make([]genstorefront.ShippingOption, len(options))
	for i, o := range options {
		genOptions[i] = genstorefront.ShippingOption{Title: o.Title, AmountMinorUnits: o.AmountMinorUnits, CurrencyCode: o.CurrencyCode}
	}
	middleware.OK(c, genOptions)
}

func (h *Handler) CreatePaymentIntent(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.createPaymentIntent.Execute(ctx, *resolved.SalesChannelID, token, toAddressInput(req.Shipping), strPtrOr(req.ShippingMethodTitle))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.PaymentIntent{
		ClientSecret: result.ClientSecret, PublishableKey: result.PublishableKey,
		AmountMinorUnits: result.AmountMinorUnits, CurrencyCode: result.CurrencyCode,
	})
}

func (h *Handler) CheckoutCart(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.checkoutCart.Execute(ctx, token, appstorefront.CheckoutInput{
		Email: string(req.Email), Billing: toAddressInput(req.Billing), Shipping: toAddressInput(req.Shipping),
		ShippingMethodTitle: strPtrOr(req.ShippingMethodTitle), PaymentIntentID: strPtrOr(req.PaymentIntentId),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.Checkout{
		OrderNumber: result.OrderNumber, TotalMinorUnits: result.TotalMinorUnits,
		CurrencyCode: result.CurrencyCode, PaymentUrl: result.PaymentURL,
	})
}

func toGenCart(result appstorefront.CartResult) *genstorefront.Cart {
	cart := result.Cart
	items := make([]genstorefront.CartItem, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = genstorefront.CartItem{
			SkuCode: item.SKUCode, ProductName: item.ProductName, Quantity: int(item.Quantity),
			PriceMinorUnits: item.PriceMinorUnits, CurrencyCode: item.CurrencyCode,
		}
	}
	return &genstorefront.Cart{
		Token: cart.Token, Items: items, SubtotalMinorUnits: cart.SubtotalMinorUnits(),
		CouponCode: cart.CouponCode, DiscountMinorUnits: result.DiscountMinorUnits,
	}
}

func toAddressInput(a genstorefront.CheckoutAddress) appstorefront.CheckoutAddressInput {
	return appstorefront.CheckoutAddressInput{
		FirstName: a.FirstName, LastName: a.LastName, Company: strPtrOr(a.Company),
		Address1: a.Address1, Address2: strPtrOr(a.Address2), City: a.City, State: strPtrOr(a.State),
		Postcode: a.Postcode, Country: a.Country, Phone: strPtrOr(a.Phone),
	}
}
