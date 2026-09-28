package events

// NewsletterSubscriberConfirmed fires when an address completes double
// opt-in -- e.g. for a welcome email automation.
type NewsletterSubscriberConfirmed struct {
	Base
	TenantID     string `json:"tenant_id"`
	SubscriberID string `json:"subscriber_id"`
}

func (NewsletterSubscriberConfirmed) EventName() string { return "newsletter.subscriber.confirmed" }

// NewsletterSubscriberErased fires when a subscriber is deleted on request
// (right to erasure). Carries no email: anything holding a copy of the
// list should drop this id.
type NewsletterSubscriberErased struct {
	Base
	TenantID     string `json:"tenant_id"`
	SubscriberID string `json:"subscriber_id"`
}

func (NewsletterSubscriberErased) EventName() string { return "newsletter.subscriber.erased" }
