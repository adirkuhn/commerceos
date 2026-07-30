package events

// IntegrationConnected is raised when a tenant successfully connects a new
// third-party integration (e.g. a WooCommerce store).
type IntegrationConnected struct {
	Base
	TenantID     string `json:"tenant_id"`
	Provider     string `json:"provider"`
	ConnectionID string `json:"connection_id"`
}

func (IntegrationConnected) EventName() string { return "integrations.connection.connected" }
