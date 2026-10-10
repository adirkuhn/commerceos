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
	// Params fill the type's translated text (notifications.Type.Render),
	// sent in the tenant's language; Subject/Body are the English
	// fallback for untranslated types.
	Params  map[string]any `json:"params,omitempty"`
	Subject string         `json:"subject"`
	Body    string         `json:"body"`
	Link    string         `json:"link"`
}

func (NotificationEmailRequested) EventName() string { return "notifications.email.requested" }
