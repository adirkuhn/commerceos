package events

// NotificationPreferencesUpdated is raised when a tenant changes which
// notifications go to the bell and/or by email.
type NotificationPreferencesUpdated struct {
	Base
	TenantID string `json:"tenant_id"`
}

func (NotificationPreferencesUpdated) EventName() string { return "notifications.preferences.updated" }
