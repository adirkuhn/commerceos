package webhooks

import "errors"

var (
	// ErrProviderUnknown means the :provider path segment doesn't match any
	// registered provider at all -- maps to 404.
	ErrProviderUnknown = errors.New("webhooks: unknown provider")
	// ErrProviderNotImplemented means the provider is registered (a valid
	// name) but has no real signature verifier wired up yet, or the tenant
	// hasn't configured a webhook secret for it -- maps to 501.
	ErrProviderNotImplemented = errors.New("webhooks: provider not implemented")
	// ErrInvalidSignature means verification ran and rejected the request
	// -- maps to 401. Deliberately generic (no detail on which check
	// failed) so it never helps an attacker iterate toward a valid forgery.
	ErrInvalidSignature = errors.New("webhooks: invalid signature")
	// ErrFallbackEventNotFound -- maps to 404 on the admin retry/abandon actions.
	ErrFallbackEventNotFound = errors.New("webhooks: fallback event not found")
	// ErrForwardRuleNotFound -- maps to 404 on the delete-rule action.
	ErrForwardRuleNotFound = errors.New("webhooks: forward rule not found")
	// ErrInvalidTargetURL -- maps to 422 on rule creation: the target must
	// be an absolute http(s) URL, never a bare host or a non-HTTP scheme
	// (this becomes an actual outbound HTTP POST -- see
	// DeliverForwardedWebhooksUseCase).
	ErrInvalidTargetURL = errors.New("webhooks: target_url must be an absolute http(s) URL")
	// ErrProviderRequired -- maps to 422 on rule creation.
	ErrProviderRequired = errors.New("webhooks: provider is required")
)
