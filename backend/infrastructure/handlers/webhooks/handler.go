// Package webhooks is the Gin adapter for the unauthenticated webhook
// ingestion route -- thin per CLAUDE.md: parse request, call the use case,
// map the result/error to a response. No business logic here.
package webhooks

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genwebhooks "github.com/adirkuhn/commerceos/api/webhooks"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
)

// receiveWebhookUseCase is appwebhooks.ReceiveWebhookUseCase narrowed to
// point of use (ISP) -- lets the handler be unit tested without a real DB,
// NATS connection, or connection repository.
type receiveWebhookUseCase interface {
	Execute(ctx context.Context, tenantID, provider string, rawBody []byte, headers map[string]string) (appwebhooks.Result, error)
}

// Handler implements genwebhooks.ServerInterface.
type Handler struct {
	receive receiveWebhookUseCase
}

func NewHandler(receive receiveWebhookUseCase) *Handler {
	return &Handler{receive: receive}
}

func (h *Handler) ReceiveWebhook(c *gin.Context, tenantID openapi_types.UUID, provider string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeWebhookError(c, http.StatusBadRequest, "invalid_body", "could not read request body")
		return
	}

	headers := map[string]string{}
	for name := range c.Request.Header {
		headers[name] = c.Request.Header.Get(name)
	}
	// MyParcel signs nothing (no HMAC/signature header) -- its verifier
	// checks a `?token=` query parameter instead, embedded in the callback
	// URL registered with MyParcel (see webhookproviders.MyParcelVerifier's
	// doc comment). Injected into the same headers map ProviderVerifier
	// already takes rather than widening that interface for one provider.
	headers["__query_token"] = c.Query("token")

	result, err := h.receive.Execute(c.Request.Context(), tenantID.String(), provider, body, headers)
	if err != nil {
		switch {
		case errors.Is(err, webhooks.ErrProviderUnknown):
			writeWebhookError(c, http.StatusNotFound, "provider_unknown", "unknown webhook provider")
		case errors.Is(err, webhooks.ErrProviderNotImplemented):
			writeWebhookError(c, http.StatusNotImplemented, "provider_not_implemented", "webhook verification not configured for this provider yet")
		case errors.Is(err, webhooks.ErrInvalidSignature):
			writeWebhookError(c, http.StatusUnauthorized, "invalid_signature", "webhook signature verification failed")
		default:
			writeWebhookError(c, http.StatusInternalServerError, "internal_error", "failed to process webhook")
		}
		return
	}

	status := http.StatusOK
	if result.FallbackUsed {
		status = http.StatusAccepted
	}
	c.JSON(status, gin.H{"data": gin.H{"event_id": result.EventID, "fallback": result.FallbackUsed}, "error": nil})
}

func writeWebhookError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message}})
}

// RegisterPublicRoutes mounts the unauthenticated webhook-ingestion route
// (matching the separate api/webhooks OpenAPI spec) -- must be called on
// the root router, outside the JWT-protected group, since third-party
// providers call this server-to-server with no bearer token. Same pattern
// as infrastructure/handlers/integrations.RegisterPublicRoutes.
func RegisterPublicRoutes(r gin.IRouter, h *Handler) {
	genwebhooks.RegisterHandlersWithOptions(r, h, genwebhooks.GinServerOptions{BaseURL: "/api/v1"})
}

// Register is the production wiring: creates the use case, then calls
// RegisterPublicRoutes.
func Register(r gin.IRouter, db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, publisher appwebhooks.Publisher, fallbackRepo webhooks.FallbackEventRepository, encryptionSecret string) {
	uc := appwebhooks.NewReceiveWebhookUseCase(db, connRepo, webhookproviders.NewRegistry(), publisher, fallbackRepo, encryptionSecret)
	h := NewHandler(uc)
	RegisterPublicRoutes(r, h)
}
