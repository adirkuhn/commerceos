package integrations

// ProviderWebhook is one webhook subscription as actually registered at
// the provider (WooCommerce or MyParcel) -- distinct from
// domain/webhooks.ForwardRule, which is CommerceOS's own fan-out
// configuration on top of an event once received. Normalizes each
// provider's own shape (WooWebhook / WebhookSubscription) into one DTO for
// the admin UI/API, see application/integrations.ListProviderWebhooksUseCase.
type ProviderWebhook struct {
	ID          string
	Provider    Provider
	Topic       string
	DeliveryURL string
	Status      string
	// ConnectionID/StoreLabel are populated by the tenant-wide aggregate
	// read (ListAllProviderWebhooksUseCase) so the admin UI can show which
	// connection each row belongs to and target Delete at the right one --
	// both empty when this came from the single-connection
	// ListProviderWebhooksUseCase, which already has connectionID in scope
	// via its own request.
	ConnectionID string
	StoreLabel   string
}
