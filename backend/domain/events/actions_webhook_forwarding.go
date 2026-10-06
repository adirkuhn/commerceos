package events

// Webhook forwarding and inbound-webhook admin actions
// (backend/application/webhooks). Domain webhook_forwarding, not webhooks:
// webhooks.> belongs to the separate inbound-webhook stream.
var (
	ForwardRuleCreated = defineAction("webhook_forwarding.forward_rule.created",
		"A rule forwarding a provider's inbound webhooks to another URL was added. data: provider, topic, target_host (the host only -- a target URL's path/query can carry secrets).")
	ForwardRuleDeleted = defineAction("webhook_forwarding.forward_rule.deleted",
		"A webhook forward rule was removed.")
	FallbackEventRepublished = defineAction("webhook_forwarding.fallback_event.republished",
		"An inbound webhook that couldn't be queued when it arrived was published again by an admin. data: provider.")
	FallbackEventAbandoned = defineAction("webhook_forwarding.fallback_event.abandoned",
		"An admin chose not to process an inbound webhook that couldn't be queued (kept for audit). data: provider.")
	FallbackEventsBulkAbandoned = defineAction("webhook_forwarding.fallback_event.bulk_abandoned",
		"An admin abandoned every inbound webhook waiting in one status at once. entity_id is the tenant. data: from_status, count.")
)
