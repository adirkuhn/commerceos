package events

// Dead-letter admin actions (backend/application/deadletters). Raised in the
// dead letter's own tenant; a dead letter without a tenant raises none.
var (
	DeadLetterRetried = defineAction("deadletters.dead_letter.retried",
		"A super-admin published a dead-lettered event again for the consumer that failed it, and removed it from the dead-letter queue. entity_id is the event ID. data: stream, sequence, subject, consumer (empty: every consumer on the subject).")
	DeadLetterDiscarded = defineAction("deadletters.dead_letter.discarded",
		"A super-admin deleted a dead-lettered event without processing it. entity_id is the event ID. data: stream, sequence, subject, consumer, error.")
)
