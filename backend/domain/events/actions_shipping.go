package events

// Shipping actions (backend/application/shipping).
var (
	ShipmentCreated = defineAction("shipping.shipment.created",
		"A shipping label was created with the carrier (MyParcel). data: sales_order_id, connection_id, carrier, external_shipment_id, barcode, package_type, insured.")
	ShipmentCancelled = defineAction("shipping.shipment.cancelled",
		"A shipment was cancelled in CommerceOS so a corrected one can be made. data: sales_order_id, deleted_on_carrier (false when the carrier refused, so its label may still exist), note.")
	FulfilmentOrderExported = defineAction("shipping.fulfilment_order.exported",
		"A sales order was exported to MyParcel as a fulfilment order. data: sales_order_id, connection_id, myparcel_uuid, status.")
	ShippingSettingsUpdated = defineAction("shipping.connection_settings.updated",
		"A shipping connection's defaults changed. entity_id is the connection id. changes: default_carrier, default_package_type, default_label_format, default_label_size, default_signature_required, default_only_recipient, insure_by_default, myparcel_account_id, myparcel_shop_id, auto_print_enabled, export_order_to_myparcel.")
	CarrierOverrideUpdated = defineAction("shipping.carrier_override.updated",
		"A carrier's per-connection option overrides were set. entity_id is the connection id. data: carrier. changes: package_type, signature_required, only_recipient, insure_by_default, tracking_url_template.")
	WebhookEventRedispatched = defineAction("shipping.webhook_event.redispatched",
		"Someone retried dispatching a stored carrier/channel webhook. data: provider, topic, status (processed or failed), error. What the webhook itself changed raises its own events.")
)
