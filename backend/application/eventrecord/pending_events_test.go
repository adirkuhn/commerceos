package eventrecord_test

// pendingEvents: files under application/ that wrote without raising an
// event when the every-action-raises-an-event rule (docs/events.md) was
// adopted, 2026-10-06. Burned down module by module; only ever shrinks --
// TestEveryWriteRaisesAnEvent fails when a file here is fixed but not
// removed, or a new file writes without an event.
var pendingEvents = []string{
	"apikeys/create_api_key.go",
	"integrations/connect_ai_provider.go",
	"integrations/disconnect_ai_connection.go",
	"integrations/disconnect_connection.go",
	"integrations/provider_webhooks.go",
	"integrations/set_webhook_secret.go",
	"integrations/set_whatsapp_ai_parsing.go",
	"integrations/set_whatsapp_test_mode.go",
	"integrations/test_connection.go",
	"integrations/update_connection_settings.go",
	"integrations/update_whatsapp_credentials.go",
	"messagecenter/create_automation_rule.go",
	"messagecenter/create_message_template.go",
	"messagecenter/delete_automation_rule.go",
	"messagecenter/delete_message_template.go",
	"messagecenter/update_automation_rule.go",
	"messagecenter/update_message_template.go",
	"newsletter/subscribe.go",
	"newsletter/unsubscribe.go",
	"scheduler/reconcile_fulfilled_orders.go",
	"scheduler/update_job_config.go",
	"webhooks/deliver_forwarded_webhooks.go",
	"webhooks/fallback_admin.go",
	"webhooks/forward_rules_admin.go",
	"webhooks/process_webhook_event.go",
	"webhooks/receive_webhook.go",
	"whatsapp/generate_payment_link.go",
	"whatsapp/handle_inbound_message.go",
	"whatsapp/list_conversations.go",
	"whatsapp/order_flow.go",
}
