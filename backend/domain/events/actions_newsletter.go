package events

// Newsletter actions (backend/application/newsletter), alongside the typed
// newsletter.subscriber.confirmed / .erased events. The email address is
// never in an event -- read the subscriber by entity_id.
var (
	NewsletterSubscriberSubscribed = defineAction("newsletter.subscriber.subscribed",
		"A shopper signed up for the newsletter and was sent a confirmation link (double opt-in, not confirmed yet). data: sales_channel_id, resubscribed (true when a previously unsubscribed address signed up again).")
	NewsletterSubscriberUnsubscribed = defineAction("newsletter.subscriber.unsubscribed",
		"A subscriber unsubscribed through their unsubscribe link.")
)
