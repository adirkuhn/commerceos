package suppliers

import "time"

type SupplierStatus string

const (
	SupplierStatusActive   SupplierStatus = "active"
	SupplierStatusInactive SupplierStatus = "inactive"
)

type Supplier struct {
	ID       string
	TenantID string
	Name     string
	Email    string
	Phone    string
	Country  string
	// LeadTimeDays is nil when unknown/not yet negotiated with this supplier.
	LeadTimeDays *int
	Status       SupplierStatus
	Notes        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
