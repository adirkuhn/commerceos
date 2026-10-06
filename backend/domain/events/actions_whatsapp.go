package events

// WhatsApp actions (backend/application/whatsapp). Message text and the
// customer's contact details are never in an event -- read the message or
// conversation by entity_id.
var (
	WhatsAppMessageReceived = defineAction("whatsapp.message.received",
		"A customer's WhatsApp message arrived and was logged. data: customer_id (null when the number isn't a known customer), webhook_event_id.")
	WhatsAppMessageSent = defineAction("whatsapp.message.sent",
		"CommerceOS replied to a customer on WhatsApp. data: customer_id.")
	WhatsAppConversationUpdated = defineAction("whatsapp.conversation.updated",
		"A WhatsApp ordering conversation moved on (cart changed, a question answered, an order placed). entity_id is the phone number. data: state, cart_item_count.")
	WhatsAppConversationDeleted = defineAction("whatsapp.conversation.deleted",
		"An admin deleted a WhatsApp conversation's message log. entity_id is the phone number.")
)
