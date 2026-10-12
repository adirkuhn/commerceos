package events

// Message Center actions (backend/application/messagecenter).
var (
	MessageTemplateCreated = defineAction("messagecenter.message_template.created",
		"A message template was created. data: name, category, channels.")
	MessageTemplateUpdated = defineAction("messagecenter.message_template.updated",
		"A message template changed (changes: name, category, channels, subject, body, enabled).")
	MessageTemplateDeleted = defineAction("messagecenter.message_template.deleted",
		"A message template was deleted. data: name.")
	AutomationRuleCreated = defineAction("messagecenter.automation_rule.created",
		"An automation rule (send a template when an event happens) was created. data: name, trigger_subject, template_id.")
	AutomationRuleUpdated = defineAction("messagecenter.automation_rule.updated",
		"An automation rule changed (changes: name, trigger_subject, conditions, template_id, channel_override, enabled).")
	AutomationRuleDeleted = defineAction("messagecenter.automation_rule.deleted",
		"An automation rule was deleted. data: name.")
	// The recipient address is never in these -- read the row by entity_id.
	SentEmailRecorded = defineAction("messagecenter.sent_email.recorded",
		"An email was sent, failed or was skipped and is now in the sent-email history. data: source, status, trigger_subject.")
	SentEmailDeleted = defineAction("messagecenter.sent_email.deleted",
		"An entry was deleted from the sent-email history.")
)
