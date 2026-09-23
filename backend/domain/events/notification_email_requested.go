package events

// NotificationEmailRequested is raised when a notification must also go
// out by email (its category has email delivery on). A consumer sends it
// through the tenant's email connection, outside the producer's
// transaction.
type NotificationEmailRequested struct {
	Base
	TenantID   string   `json:"tenant_id"`
	Type       string   `json:"type"`
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	Link       string   `json:"link"`
}

func (NotificationEmailRequested) EventName() string { return "notifications.email.requested" }
