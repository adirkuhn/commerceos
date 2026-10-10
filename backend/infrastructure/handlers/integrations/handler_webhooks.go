package integrations

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) SetWebhookSecret(c *gin.Context, id genintegrations.Id) {
	// Body is optional (backward compatible with the generate-only flow) --
	// a missing/empty body just means "generate one," not a bind error.
	var req genintegrations.SetWebhookSecretRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
	}
	providedSecret := ""
	if req.Secret != nil {
		providedSecret = *req.Secret
	}
	result, err := h.setWebhookSecret.Execute(c.Request.Context(), id, providedSecret)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"connection_id": result.ConnectionID, "webhook_secret": result.PlaintextSecret})
}

func (h *Handler) ListWebhookFallbackEvents(c *gin.Context, params genintegrations.ListWebhookFallbackEventsParams) {
	status := webhooksDomain.FallbackStatusPendingRepublish
	if params.Status != nil {
		status = webhooksDomain.FallbackStatus(*params.Status)
	}
	page, perPage := 0, 0
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	result, err := h.listWebhookFallbackEvents.Execute(c.Request.Context(), status, page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookFallbackEvent, len(result.Events))
	for i, e := range result.Events {
		id, eventID, provider := e.ID, e.EventID, e.Provider
		st := genintegrations.WebhookFallbackEventStatus(e.Status)
		receivedAt := e.ReceivedAt
		out[i] = genintegrations.WebhookFallbackEvent{
			Id: &id, EventId: &eventID, Provider: &provider, Status: &st, ReceivedAt: &receivedAt,
		}
	}
	total := int(result.Total)
	c.JSON(http.StatusOK, genintegrations.WebhookFallbackEventListResponse{Data: &out, Total: &total})
}

func (h *Handler) ListWebhookForwardRules(c *gin.Context) {
	rules, err := h.listWebhookForwardRules.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookForwardRule, len(rules))
	for i, r := range rules {
		id, provider, topic, targetURL, active, createdAt := r.ID, r.Provider, r.Topic, r.TargetURL, r.Active, r.CreatedAt
		out[i] = genintegrations.WebhookForwardRule{
			Id: &id, Provider: &provider, Topic: &topic, TargetUrl: &targetURL, Active: &active, CreatedAt: &createdAt,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) CreateWebhookForwardRule(c *gin.Context) {
	var req genintegrations.CreateWebhookForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	topic := ""
	if req.Topic != nil {
		topic = *req.Topic
	}
	rule, err := h.createWebhookForwardRule.Execute(c.Request.Context(), req.Provider, topic, req.TargetUrl)
	if err != nil {
		if errors.Is(err, webhooksDomain.ErrInvalidTargetURL) || errors.Is(err, webhooksDomain.ErrProviderRequired) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
		if err := mapError(c, err); err != nil {
			return
		}
	}
	middleware.OK(c, genintegrations.WebhookForwardRule{
		Id: &rule.ID, Provider: &rule.Provider, Topic: &rule.Topic, TargetUrl: &rule.TargetURL,
		Active: &rule.Active, CreatedAt: &rule.CreatedAt,
	})
}

func (h *Handler) DeleteWebhookForwardRule(c *gin.Context, id genintegrations.Id) {
	err := h.deleteWebhookForwardRule.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListWebhookForwardDeliveries(c *gin.Context, params genintegrations.ListWebhookForwardDeliveriesParams) {
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	deliveries, err := h.listWebhookForwardDeliveries.Execute(c.Request.Context(), limit)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookForwardDelivery, len(deliveries))
	for i, d := range deliveries {
		id, ruleID, provider, topic, targetURL := d.ID, d.RuleID, d.Provider, d.Topic, d.TargetURL
		status := genintegrations.WebhookForwardDeliveryStatus(d.Status)
		attempts, lastError, nextAttemptAt, createdAt := d.Attempts, d.LastError, d.NextAttemptAt, d.CreatedAt
		out[i] = genintegrations.WebhookForwardDelivery{
			Id: &id, RuleId: &ruleID, Provider: &provider, Topic: &topic, TargetUrl: &targetURL,
			Status: &status, Attempts: &attempts, LastError: &lastError,
			NextAttemptAt: &nextAttemptAt, DeliveredAt: d.DeliveredAt, CreatedAt: &createdAt,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) ListWebhookEvents(c *gin.Context, params genintegrations.ListWebhookEventsParams) {
	provider := ""
	if params.Provider != nil {
		provider = *params.Provider
	}
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	events, err := h.listWebhookEvents.Execute(c.Request.Context(), provider, limit)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookEvent, len(events))
	for i, e := range events {
		out[i] = webhookEventToResponse(e)
	}
	middleware.OK(c, out)
}

func (h *Handler) RetryWebhookEventDispatch(c *gin.Context, id genintegrations.Id) {
	event, err := h.retryWebhookEventDispatch.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, webhookEventToResponse(*event))
}

func webhookEventToResponse(e webhooksDomain.WebhookEvent) genintegrations.WebhookEvent {
	id, eventID, provider, topic := e.ID, e.EventID, e.Provider, e.Topic
	receivedAt, processedAt := e.ReceivedAt, e.ProcessedAt
	status := genintegrations.WebhookEventStatus(e.Status)
	dispatchError := e.DispatchError
	var payload any
	if err := json.Unmarshal(e.Payload, &payload); err != nil && len(e.Payload) > 0 {
		log.Printf("integrationsHandler: webhook event %s: corrupt payload, omitting: %v", e.ID, err)
	}
	return genintegrations.WebhookEvent{
		Id: &id, EventId: &eventID, Provider: &provider, Topic: &topic,
		Payload: payload, ReceivedAt: &receivedAt, ProcessedAt: &processedAt,
		Status: &status, DispatchError: &dispatchError,
	}
}

func (h *Handler) ListAllProviderWebhooks(c *gin.Context) {
	result, err := h.listAllProviderWebhooks.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.ProviderWebhook, len(result.Webhooks))
	for i, hook := range result.Webhooks {
		hookID, provider, topic, deliveryURL, status := hook.ID, string(hook.Provider), hook.Topic, hook.DeliveryURL, hook.Status
		connectionID, storeLabel := hook.ConnectionID, hook.StoreLabel
		out[i] = genintegrations.ProviderWebhook{
			Id: &hookID, Provider: &provider, Topic: &topic, DeliveryUrl: &deliveryURL, Status: &status,
			ConnectionId: &connectionID, StoreLabel: &storeLabel,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) ListAvailableWebhookTopics(c *gin.Context) {
	topics, err := h.listAvailableWebhookTopics.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	allEvents := genintegrations.TopicsByProvider(topics.AllEvents)
	activeWebhooks := genintegrations.TopicsByProvider(topics.ActiveWebhooks)
	middleware.OK(c, genintegrations.AvailableWebhookTopics{AllEvents: &allEvents, ActiveWebhooks: &activeWebhooks})
}

func (h *Handler) ListProviderWebhooks(c *gin.Context, id genintegrations.Id) {
	hooks, err := h.listProviderWebhooks.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, integrationsDomain.ErrProviderNotSupported) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
		if errors.Is(err, integrationsDomain.ErrProviderWebhookRequestFailed) {
			middleware.FailErr(c, http.StatusBadGateway, "PROVIDER_REQUEST_FAILED", err)
			return
		}
		if err := mapError(c, err); err != nil {
			return
		}
	}
	out := make([]genintegrations.ProviderWebhook, len(hooks))
	for i, hook := range hooks {
		hookID, provider, topic, deliveryURL, status := hook.ID, string(hook.Provider), hook.Topic, hook.DeliveryURL, hook.Status
		out[i] = genintegrations.ProviderWebhook{Id: &hookID, Provider: &provider, Topic: &topic, DeliveryUrl: &deliveryURL, Status: &status}
	}
	middleware.OK(c, out)
}

func (h *Handler) CreateProviderWebhook(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.CreateProviderWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	hook, err := h.createProviderWebhook.Execute(c.Request.Context(), id, req.Topic)
	if err != nil {
		if errors.Is(err, integrationsDomain.ErrWebhookSecretNotSet) || errors.Is(err, integrationsDomain.ErrProviderNotSupported) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
		if errors.Is(err, integrationsDomain.ErrProviderWebhookRequestFailed) {
			middleware.FailErr(c, http.StatusBadGateway, "PROVIDER_REQUEST_FAILED", err)
			return
		}
		if err := mapError(c, err); err != nil {
			return
		}
	}
	provider := string(hook.Provider)
	middleware.OK(c, genintegrations.ProviderWebhook{
		Id: &hook.ID, Provider: &provider, Topic: &hook.Topic, DeliveryUrl: &hook.DeliveryURL, Status: &hook.Status,
	})
}

func (h *Handler) DeleteProviderWebhook(c *gin.Context, id genintegrations.Id, webhookId genintegrations.WebhookId) {
	err := h.deleteProviderWebhook.Execute(c.Request.Context(), id, webhookId)
	if err != nil {
		if errors.Is(err, integrationsDomain.ErrProviderWebhookRequestFailed) {
			middleware.FailErr(c, http.StatusBadGateway, "PROVIDER_REQUEST_FAILED", err)
			return
		}
		if err := mapError(c, err); err != nil {
			return
		}
	}
	middleware.OK(c, nil)
}

func (h *Handler) RetryWebhookFallbackEvent(c *gin.Context, id genintegrations.Id) {
	err := h.retryWebhookFallbackEvent.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, webhooksDomain.ErrFallbackEventNotFound) {
			middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
			return
		}
		if errors.Is(err, webhooksDomain.ErrPayloadTooLarge) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "PAYLOAD_TOO_LARGE", err)
			return
		}
		middleware.FailErr(c, http.StatusBadGateway, "PUBLISH_FAILED", err)
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) AbandonWebhookFallbackEvent(c *gin.Context, id genintegrations.Id) {
	err := h.abandonWebhookFallbackEvent.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) AbandonAllWebhookFallbackEvents(c *gin.Context, params genintegrations.AbandonAllWebhookFallbackEventsParams) {
	status := webhooksDomain.FallbackStatusPendingRepublish
	if params.Status != nil {
		status = webhooksDomain.FallbackStatus(*params.Status)
	}
	count, err := h.abandonAllWebhookFallbackEvents.Execute(c.Request.Context(), status)
	if err := mapError(c, err); err != nil {
		return
	}
	abandoned := int(count)
	c.JSON(http.StatusOK, genintegrations.AbandonAllWebhookFallbackEventsResponse{
		Data: &struct {
			Abandoned *int `json:"abandoned,omitempty"`
		}{Abandoned: &abandoned},
	})
}
