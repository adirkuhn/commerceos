package webhooks

import "time"

// ForwardRule is a tenant's own "also send this to me" subscription on top
// of CommerceOS's normal webhook ingestion -- the fan-out configured from
// the Integrations page's Webhooks settings (e.g. "myparcel
// shipment_status_change -> https://shop.example.com/wc-myparcel-webhook").
// Multiple rules can share the same Provider+Topic to fan out to several
// URLs; see ForwardDelivery for the one-row-per-(rule,event) delivery
// record this drives.
type ForwardRule struct {
	ID string
	// Topic == "" matches every event received for Provider -- needed for
	// a provider (MyParcel today) that doesn't send a per-delivery topic;
	// see ProcessWebhookEventUseCase's topicHeader doc comment.
	TenantID  string
	Provider  string
	Topic     string
	TargetURL string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ForwardDeliveryStatus string

const (
	ForwardDeliveryStatusPending   ForwardDeliveryStatus = "pending"
	ForwardDeliveryStatusDelivered ForwardDeliveryStatus = "delivered"
	ForwardDeliveryStatusFailed    ForwardDeliveryStatus = "failed"
)

// ForwardDelivery is one attempt (in progress or concluded) to relay a
// received webhook's exact raw payload to one ForwardRule's TargetURL.
// Written inside the same transaction as the triggering WebhookEvent (see
// ProcessWebhookEventUseCase) so the intent to forward is durable even if
// the HTTP call itself later fails -- DeliverForwardedWebhooksUseCase is
// what actually drains Pending rows.
type ForwardDelivery struct {
	ID             string
	TenantID       string
	RuleID         string
	WebhookEventID string
	Provider       string
	Topic          string
	TargetURL      string
	Payload        []byte
	Status         ForwardDeliveryStatus
	Attempts       int
	LastError      string
	NextAttemptAt  time.Time
	DeliveredAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
