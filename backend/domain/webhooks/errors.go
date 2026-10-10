package webhooks

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	// ErrProviderUnknown means the :provider path segment doesn't match any
	// registered provider at all -- maps to 404.
	ErrProviderUnknown = errs.New("WEBHOOK_PROVIDER_UNKNOWN", "webhooks: unknown provider")
	// ErrProviderNotImplemented means the provider is registered (a valid
	// name) but has no real signature verifier wired up yet, or the tenant
	// hasn't configured a webhook secret for it -- maps to 501.
	ErrProviderNotImplemented = errs.New("WEBHOOK_PROVIDER_NOT_IMPLEMENTED", "webhooks: provider not implemented")
	// ErrInvalidSignature means verification ran and rejected the request
	// -- maps to 401. Deliberately generic (no detail on which check
	// failed) so it never helps an attacker iterate toward a valid forgery.
	ErrInvalidSignature = errs.New("WEBHOOK_INVALID_SIGNATURE", "webhooks: invalid signature")
	// ErrFallbackEventNotFound -- maps to 404 on the admin retry/abandon actions.
	ErrFallbackEventNotFound = errs.New("WEBHOOK_FALLBACK_EVENT_NOT_FOUND", "webhooks: fallback event not found")
	// ErrForwardRuleNotFound -- maps to 404 on the delete-rule action.
	ErrForwardRuleNotFound = errs.New("WEBHOOK_FORWARD_RULE_NOT_FOUND", "webhooks: forward rule not found")
	// ErrInvalidTargetURL -- maps to 422 on rule creation: the target must
	// be an absolute http(s) URL, never a bare host or a non-HTTP scheme
	// (this becomes an actual outbound HTTP POST -- see
	// DeliverForwardedWebhooksUseCase).
	ErrInvalidTargetURL = errs.New("WEBHOOK_INVALID_TARGET_URL", "webhooks: target_url must be an absolute http(s) URL")
	// ErrProviderRequired -- maps to 422 on rule creation.
	ErrProviderRequired = errs.New("WEBHOOK_PROVIDER_REQUIRED", "webhooks: provider is required")
	// ErrPayloadTooLarge means a webhook payload exceeds the NATS server's
	// max_payload -- publishing (or re-publishing via the fallback retry
	// action) will fail identically on every attempt with these exact bytes,
	// so it's caught before the publish call instead of surfacing NATS's own
	// opaque "maximum payload exceeded" error. Maps to 422 on the fallback
	// retry admin action.
	ErrPayloadTooLarge = errs.New("WEBHOOK_PAYLOAD_TOO_LARGE", "webhooks: payload exceeds NATS max payload")
)
