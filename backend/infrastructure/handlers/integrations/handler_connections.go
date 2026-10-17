package integrations

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	gencallback "github.com/adirkuhn/commerceos/api/integrationscallback"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Route handlers (api/integrations — protected) ────────────────────────────

func (h *Handler) ListCatalog(c *gin.Context) {
	middleware.OK(c, []genintegrations.CatalogEntry{
		{Provider: string(integrationsDomain.ProviderWooCommerce), Name: "WooCommerce"},
		{Provider: "ai", Name: "AI"},
	})
}

func (h *Handler) ListConnections(c *gin.Context) {
	conns, err := h.listConnections.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.Connection, len(conns))
	for i, conn := range conns {
		resp[i] = toGenConnection(conn)
	}
	middleware.OK(c, resp)
}

func (h *Handler) GetAIConnection(c *gin.Context) {
	conn, err := h.getAIConnection.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	if conn == nil {
		middleware.OK(c, nil)
		return
	}
	resp := toGenAIConnection(*conn)
	middleware.OK(c, &resp)
}

func (h *Handler) ConnectAIProvider(c *gin.Context) {
	var req genintegrations.ConnectAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	in := appintegrations.ConnectAIProviderInput{Provider: integrationsDomain.AIProvider(req.Provider)}
	if req.ApiKey != nil {
		in.APIKey = *req.ApiKey
	}
	if req.BaseUrl != nil {
		in.BaseURL = *req.BaseUrl
	}
	if req.Model != nil {
		in.Model = *req.Model
	}
	if req.FallbackModels != nil {
		in.FallbackModels = *req.FallbackModels
	}
	conn, err := h.connectAIProvider.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenAIConnection(*conn)
	middleware.OK(c, &resp)
}

func (h *Handler) DisconnectAIProvider(c *gin.Context) {
	if err := mapError(c, h.disconnectAIConnection.Execute(c.Request.Context())); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) StartWooConnect(c *gin.Context) {
	var req genintegrations.StartWooConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	result, err := h.startWooConnect.Execute(c.Request.Context(), appintegrations.StartWooConnectInput{StoreURL: req.StoreUrl})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"authorize_url": result.AuthorizeURL})
}

func (h *Handler) ManualConnectWooCommerce(c *gin.Context) {
	var req genintegrations.ManualConnectWooCommerceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.StoreUrl == "" || req.ConsumerKey == "" || req.ConsumerSecret == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	conn, err := h.manualConnectWoo.Execute(c.Request.Context(), appintegrations.ManualConnectWooCommerceInput{
		StoreURL:       req.StoreUrl,
		ConsumerKey:    req.ConsumerKey,
		ConsumerSecret: req.ConsumerSecret,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenConnection(*conn)
	middleware.OK(c, resp)
}

func (h *Handler) DeleteConnection(c *gin.Context, id genintegrations.Id) {
	err := h.disconnect.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) TestConnection(c *gin.Context, id genintegrations.Id) {
	conn, err := h.testConnection.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenConnection(*conn)
	middleware.OK(c, resp)
}

func (h *Handler) UpdateConnectionSettings(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpdateConnectionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	conn, err := h.updateConnectionSettings.Execute(c.Request.Context(), appintegrations.UpdateConnectionSettingsInput{
		ID: id, DefaultCurrency: req.DefaultCurrency, DefaultWarehouseID: req.DefaultWarehouseId,
		BolEconomicOperatorID: req.BolEconomicOperatorId, BolDeliveryCode: req.BolDeliveryCode,
		BolShippingCostMinorUnits: req.BolShippingCostMinorUnits,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) UpdateWhatsAppCredentials(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpdateWhatsAppCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	apiKey := ""
	if req.ApiKey != nil {
		apiKey = *req.ApiKey
	}
	conn, err := h.updateWhatsAppCredentials.Execute(c.Request.Context(), id, appintegrations.UpdateWhatsAppCredentialsInput{
		BaseURL: req.BaseUrl, InstanceName: req.InstanceName, APIKey: apiKey,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) SetWhatsAppTestMode(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.SetWhatsAppTestModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	testPhone := ""
	if req.TestPhoneE164 != nil {
		testPhone = *req.TestPhoneE164
	}
	conn, err := h.setWhatsAppTestMode.Execute(c.Request.Context(), id, req.Enabled, testPhone)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) SetWhatsAppAIParsing(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.SetWhatsAppAIParsingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	conn, err := h.setWhatsAppAIParsing.Execute(c.Request.Context(), id, req.Enabled)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) ConnectBol(c *gin.Context) {
	var req genintegrations.ConnectBolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.ClientId == "" || req.ClientSecret == "" || req.EconomicOperatorId == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	conn, err := h.connectBol.Execute(c.Request.Context(), appintegrations.ConnectBolInput{
		ClientID: req.ClientId, ClientSecret: req.ClientSecret, EconomicOperatorID: req.EconomicOperatorId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) ConnectWhatsapp(c *gin.Context) {
	var req genintegrations.ConnectWhatsAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.BaseUrl == "" || req.InstanceName == "" || req.ApiKey == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	result, err := h.connectWhatsapp.Execute(c.Request.Context(), appintegrations.ConnectWhatsAppInput{
		BaseURL: req.BaseUrl, InstanceName: req.InstanceName, APIKey: req.ApiKey,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, struct {
		Connection genintegrations.Connection `json:"connection"`
		WebhookURL string                     `json:"webhook_url"`
	}{Connection: toGenConnection(result.Connection), WebhookURL: result.WebhookURL})
}

func (h *Handler) ConnectStripe(c *gin.Context) {
	var req genintegrations.ConnectStripeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.SecretKey == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	publishableKey := ""
	if req.PublishableKey != nil {
		publishableKey = *req.PublishableKey
	}
	result, err := h.connectStripe.Execute(c.Request.Context(), appintegrations.ConnectStripeInput{SecretKey: req.SecretKey, PublishableKey: publishableKey})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, struct {
		Connection genintegrations.Connection `json:"connection"`
		WebhookURL string                     `json:"webhook_url"`
	}{Connection: toGenConnection(*result.Connection), WebhookURL: result.WebhookURL})
}

func (h *Handler) ConnectRevolut(c *gin.Context) {
	var req genintegrations.ConnectRevolutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.ApiKey == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	result, err := h.connectRevolut.Execute(c.Request.Context(), appintegrations.ConnectRevolutInput{APIKey: req.ApiKey})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, struct {
		Connection genintegrations.Connection `json:"connection"`
		WebhookURL string                     `json:"webhook_url"`
	}{Connection: toGenConnection(*result.Connection), WebhookURL: result.WebhookURL})
}

func (h *Handler) ListPaymentConnections(c *gin.Context) {
	conns, err := h.listPaymentConnections.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.Connection, len(conns))
	for i, conn := range conns {
		out[i] = toGenConnection(conn)
	}
	middleware.OK(c, out)
}

// ── Route handler (api/integrationscallback — public) ────────────────────────

// WooCallback is unauthenticated — WooCommerce itself calls this
// server-to-server after the merchant approves on their own store. Must
// return 200 quickly: WooCommerce deletes the newly created API key on any
// non-2xx response.
// WooCallback is the single unauthenticated, server-to-server route Woo
// calls after a merchant approves the connect flow -- there's no gin
// access-log middleware mounted anywhere in this app (see cmd/api/main.go),
// and mapError only logs truly unhandled errors, so a known failure here
// (e.g. a state-signature mismatch from JWT_SECRET changing between the
// connect attempt starting and this callback landing) would otherwise
// leave zero trace of ever having been hit. Always log the outcome --
// low-volume, security-sensitive endpoint, worth the audit trail
// unconditionally. Never logs req.ConsumerKey/ConsumerSecret themselves.
func (h *Handler) WooCallback(c *gin.Context) {
	var req gencallback.WooCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("wooCallback: bind request: %v", err)
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	conn, err := h.completeWooConnect.Execute(c.Request.Context(), appintegrations.CompleteWooConnectInput{
		State:          req.UserId,
		ConsumerKey:    req.ConsumerKey,
		ConsumerSecret: req.ConsumerSecret,
	})
	if err != nil {
		log.Printf("wooCallback: %v", err)
		mapError(c, err)
		return
	}
	log.Printf("wooCallback: connected connection_id=%s", conn.ID)
	middleware.OK(c, gin.H{"connection_id": conn.ID})
}

// ── Domain → generated-type mapper ───────────────────────────────────────────

func toGenConnection(c integrationsDomain.Connection) genintegrations.Connection {
	currency := c.DefaultCurrency
	var whatsappInstanceName *string
	if c.Provider == integrationsDomain.ProviderWhatsApp {
		whatsappInstanceName = &c.ConsumerKey
	}
	return genintegrations.Connection{
		Id:                        c.ID,
		Provider:                  string(c.Provider),
		StoreUrl:                  c.StoreURL.String(),
		Status:                    genintegrations.ConnectionStatus(c.Status),
		DefaultCurrency:           &currency,
		DefaultWarehouseId:        c.DefaultWarehouseID,
		BolEconomicOperatorId:     c.BolEconomicOperatorID,
		BolDeliveryCode:           c.BolDeliveryCode,
		BolShippingCostMinorUnits: c.BolShippingCostMinorUnits,
		WhatsappTestModeEnabled:   &c.WhatsAppTestModeEnabled,
		WhatsappTestPhoneE164:     c.WhatsAppTestPhoneE164,
		WhatsappAiParsingEnabled:  &c.WhatsAppAIParsingEnabled,
		WhatsappInstanceName:      whatsappInstanceName,
		CreatedAt:                 c.CreatedAt,
	}
}

func toGenAIConnection(c integrationsDomain.AIConnection) genintegrations.AIConnection {
	resp := genintegrations.AIConnection{
		Id: c.ID, Provider: genintegrations.AIProvider(c.Provider),
		Status: genintegrations.ConnectionStatus(c.Status), CreatedAt: c.CreatedAt,
	}
	if c.BaseURL != "" {
		resp.BaseUrl = &c.BaseURL
	}
	if c.Model != "" {
		resp.Model = &c.Model
	}
	if c.FallbackModels != "" {
		resp.FallbackModels = &c.FallbackModels
	}
	return resp
}
