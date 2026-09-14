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

// maxWebhookBodyBytes matches nats/nats.conf's max_payload -- anything that
// clears this cap is guaranteed to clear NATS too, so an oversized body
// (a WhatsApp/Evolution API media message embeds the full file as inline
// base64 -- 44.5MB seen in practice, and nothing downstream ever reads
// those bytes) is rejected here, before it's ever buffered, published, or
// durably written to the fallback table as a row that can never succeed.
const maxWebhookBodyBytes = 8 * 1024 * 1024

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
	// MyParcel signs nothing (no HMAC/signature header) -- its verifier
	// checks this token instead. Query-string form, kept for callback URLs
	// registered before ReceiveWebhookWithToken's path-segment form existed.
	h.receiveWithToken(c, tenantID, provider, c.Query("token"))
}

// ReceiveWebhookWithToken is ReceiveWebhook for a provider whose token
// travels as a path segment instead of a `?token=` query string (see
// api/webhooks/openapi.yaml's doc comment on this route) -- currently just
// MyParcel, registered via this shape from here on.
func (h *Handler) ReceiveWebhookWithToken(c *gin.Context, tenantID openapi_types.UUID, provider string, token string) {
	h.receiveWithToken(c, tenantID, provider, token)
}

func (h *Handler) receiveWithToken(c *gin.Context, tenantID openapi_types.UUID, provider string, token string) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeWebhookError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds the maximum accepted size")
			return
		}
		writeWebhookError(c, http.StatusBadRequest, "invalid_body", "could not read request body")
		return
	}

	headers := map[string]string{}
	for name := range c.Request.Header {
		headers[name] = c.Request.Header.Get(name)
	}
	// Injected into the same headers map ProviderVerifier already takes
	// rather than widening that interface for one provider (see
	// webhookproviders.MyParcelVerifier's doc comment).
	headers["__query_token"] = token

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
