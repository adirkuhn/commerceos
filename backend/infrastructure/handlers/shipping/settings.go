package shipping

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func (h *Handler) ConnectMyParcel(c *gin.Context) {
	var req genshipping.ConnectMyParcelJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	conn, err := h.connect.Execute(c.Request.Context(), appintegrations.ConnectMyParcelInput{APIKey: req.ApiKey})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"connection_id": conn.ID, "provider": string(conn.Provider), "status": string(conn.Status),
	}, "error": nil})
}

func (h *Handler) RegisterMyParcelWebhook(c *gin.Context, connectionId genshipping.ConnectionId) {
	if err := h.registerWebhook.Execute(c.Request.Context(), connectionId.String()); err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}, "error": nil})
}

func (h *Handler) GetShippingSettings(c *gin.Context, connectionId genshipping.ConnectionId) {
	result, err := h.getSettings.Execute(c.Request.Context(), connectionId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"settings": settingsToResponse(result.Settings), "carrier_overrides": overridesToResponse(result.CarrierOverrides),
	}, "error": nil})
}

func (h *Handler) UpdateShippingSettings(c *gin.Context, connectionId genshipping.ConnectionId) {
	var req genshipping.UpdateShippingSettingsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	onlyRecipient, sig := false, false
	if req.DefaultOnlyRecipient != nil {
		onlyRecipient = *req.DefaultOnlyRecipient
	}
	if req.DefaultSignatureRequired != nil {
		sig = *req.DefaultSignatureRequired
	}
	autoPrint := false
	if req.AutoPrintEnabled != nil {
		autoPrint = *req.AutoPrintEnabled
	}
	insureByDefault := false
	if req.InsureByDefault != nil {
		insureByDefault = *req.InsureByDefault
	}
	labelSize := ""
	if req.DefaultLabelSize != nil {
		labelSize = string(*req.DefaultLabelSize)
	}
	exportOrder := false
	if req.ExportOrderToMyparcel != nil {
		exportOrder = *req.ExportOrderToMyparcel
	}
	settings, err := h.updateSettings.Execute(c.Request.Context(), appshipping.UpdateShippingSettingsInput{
		ConnectionID: connectionId.String(), DefaultCarrier: shippingDomain.Carrier(req.DefaultCarrier),
		DefaultPackageType: shippingDomain.PackageType(req.DefaultPackageType), DefaultLabelFormat: string(req.DefaultLabelFormat),
		DefaultLabelSize:         labelSize,
		DefaultSignatureRequired: sig, DefaultOnlyRecipient: onlyRecipient,
		InsureByDefault:   insureByDefault,
		MyParcelAccountID: req.MyparcelAccountId, MyParcelShopID: req.MyparcelShopId, AutoPrintEnabled: autoPrint,
		ExportOrderToMyParcel: exportOrder,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settingsToResponse(settings), "error": nil})
}

func (h *Handler) UpsertCarrierOverride(c *gin.Context, connectionId genshipping.ConnectionId, carrier int) {
	var req genshipping.UpsertCarrierOverrideJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	var packageType *shippingDomain.PackageType
	if req.PackageType != nil {
		pt := shippingDomain.PackageType(*req.PackageType)
		packageType = &pt
	}
	override, err := h.upsertOverride.Execute(c.Request.Context(), appshipping.UpsertCarrierOverrideInput{
		ConnectionID: connectionId.String(), Carrier: shippingDomain.Carrier(carrier),
		PackageType: packageType, SignatureRequired: req.SignatureRequired, OnlyRecipient: req.OnlyRecipient,
		InsureByDefault: req.InsureByDefault, TrackingURLTemplate: req.TrackingUrlTemplate,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": overrideToResponse(*override), "error": nil})
}

func settingsToResponse(s *shippingDomain.ShippingConnectionSettings) *genshipping.ShippingConnectionSettings {
	if s == nil {
		return nil
	}
	carrier, pkgType := int(s.DefaultCarrier), int(s.DefaultPackageType)
	format := genshipping.ShippingConnectionSettingsDefaultLabelFormat(s.DefaultLabelFormat)
	resp := &genshipping.ShippingConnectionSettings{
		ConnectionId: &s.ConnectionID, DefaultCarrier: &carrier, DefaultPackageType: &pkgType,
		DefaultLabelFormat: &format, DefaultSignatureRequired: &s.DefaultSignatureRequired, DefaultOnlyRecipient: &s.DefaultOnlyRecipient,
		InsureByDefault:   &s.InsureByDefault,
		MyparcelAccountId: s.MyParcelAccountID, MyparcelShopId: s.MyParcelShopID, AutoPrintEnabled: &s.AutoPrintEnabled,
		ExportOrderToMyparcel: &s.ExportOrderToMyParcel,
	}
	if s.DefaultLabelSize != "" {
		size := genshipping.ShippingConnectionSettingsDefaultLabelSize(s.DefaultLabelSize)
		resp.DefaultLabelSize = &size
	}
	return resp
}

func overrideToResponse(o shippingDomain.ShippingCarrierOverride) genshipping.ShippingCarrierOverride {
	carrier := int(o.Carrier)
	resp := genshipping.ShippingCarrierOverride{
		ConnectionId: &o.ConnectionID, Carrier: &carrier, SignatureRequired: o.SignatureRequired, OnlyRecipient: o.OnlyRecipient,
		InsureByDefault: o.InsureByDefault, TrackingUrlTemplate: o.TrackingURLTemplate,
	}
	if o.PackageType != nil {
		pt := int(*o.PackageType)
		resp.PackageType = &pt
	}
	return resp
}

func overridesToResponse(overrides []shippingDomain.ShippingCarrierOverride) []genshipping.ShippingCarrierOverride {
	out := make([]genshipping.ShippingCarrierOverride, len(overrides))
	for i, o := range overrides {
		out[i] = overrideToResponse(o)
	}
	return out
}
