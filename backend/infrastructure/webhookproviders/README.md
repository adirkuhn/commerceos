# Webhook ingestion — how it works, how to extend it

Two processes, split so a slow/misbehaving vendor payload never blocks the
next one:

- **Hot path** (`cmd/api`, unauthenticated `POST /webhooks/v1/:tenant_id/:provider`,
  `application/webhooks.ReceiveWebhookUseCase`): verify signature → publish to
  JetStream stream `COMMERCEOS_WEBHOOKS` (subject `webhooks.received.{provider}`)
  with up to 3 in-request retries → on total failure, durably save to the
  `webhook_fallback_events` table instead of losing the request. Never does
  real business logic, never touches anything but the signature check and
  the publish/fallback write.
- **Cold path** (`cmd/webhook-processor`, a separate deployable, queue-group
  JetStream consumer — see `docker-compose.yml`'s `webhook-processor` service):
  dedupe by `EventID`, persist to `webhook_events`, ack. Currently *only*
  proves the event was received — no per-topic business rule dispatch yet
  (see "Wiring real processing" below).

## Adding a new provider's signature verification

Providers currently stubbed (return 501, not verified — see
`stub_verifier.go`'s doc comment for why they weren't guessed at):
Bol, Moneybird, Shopify, Lightspeed, MyParcel, Sendcloud.

1. Implement `domain/webhooks.ProviderVerifier` in a new file here,
   `<provider>_verifier.go` (one file per provider — see
   `woocommerce_verifier.go` for the reference implementation):
   ```go
   type ProviderVerifier interface {
       // Verify checks rawBody/headers against secret (the tenant's
       // decrypted per-provider webhook secret). Return
       // webhooks.ErrInvalidSignature on any mismatch.
       Verify(rawBody []byte, headers map[string]string, secret string) error
       // EventID returns a stable per-delivery id from headers if the
       // provider sends one, else "" to fall back to a raw-body hash
       // (ReceiveWebhookUseCase.Execute handles the fallback — don't
       // duplicate that here).
       EventID(headers map[string]string, rawBody []byte) string
   }
   ```
   Get the exact header name, algorithm, and encoding from that vendor's
   own webhook docs — don't guess or infer from a similar vendor. Use
   `hmac.Equal` (or equivalent constant-time compare) for any signature
   comparison, never `==` — see `woocommerce_verifier.go`'s comment on why.
2. Register it in `registry.go`: add `Provider<Name> = "<name>"` (the
   registry key doubles as the `:provider` URL segment) and wire
   `Provider<Name>: New<Name>Verifier()` into `NewRegistry()`, then remove
   that name from `stubProviders`.
3. Test it the way `woocommerce_verifier_test.go` does: valid signature,
   wrong secret, tampered body, missing header — all four cases.
4. A tenant still can't receive real webhooks from this provider until
   they provision a secret for it — that's the "Webhooks" tab on the
   provider's connection settings page in the frontend (see
   `WooCommerceSyncSettingsPage.tsx`'s Webhooks tab for the pattern;
   `SetWebhookSecretUseCase` is provider-agnostic already, no backend
   change needed there).

## Wiring real processing (per-topic business rules)

`ProcessWebhookEventUseCase.Execute` (`application/webhooks/process_webhook_event.go`)
only persists the envelope + dedupes today — deliberately, to avoid
half-implementing rules nobody asked for yet. When a topic needs a real
effect (e.g. WooCommerce's `order.updated` should actually update the
matching sales order), add the dispatch there: branch on
`envelope.Provider` + the topic (WooCommerce sends it in the
`X-Wc-Webhook-Topic` header, already captured as `WebhookEvent.Topic` —
treat it as informational only, per that field's doc comment, never as a
verified/trusted routing key on its own) and call the appropriate existing
use case (e.g. OMS's order-update use case) inside the same
`tenant.WithTenant` transaction that saves the `WebhookEvent` row, so the
"received" record and the actual effect commit atomically.

## Fallback / admin surface

`webhook_fallback_events` rows (publish failed 3x) show up in the
Integrations page's undelivered-webhooks table with Retry (re-attempt the
JetStream publish now) and Abandon (soft-remove, keeps the audit row) —
see `application/webhooks/fallback_admin.go` for the use cases and
`frontend/src/pages/Integrations.tsx` for the UI. Gated to
`customer_owner`/`customer_admin`/`super_admin` client-side (this page has
no client-side RBAC elsewhere today — this was a deliberate exception, see
commit `d73623a`).
