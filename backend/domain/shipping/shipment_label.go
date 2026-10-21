package shipping

import "time"

// ShipmentLabel is a stored label file (see ShipmentLabelRepository).
type ShipmentLabel struct {
	ShipmentID  string
	TenantID    string
	ContentType string
	Data        []byte
	CreatedAt   time.Time
}
