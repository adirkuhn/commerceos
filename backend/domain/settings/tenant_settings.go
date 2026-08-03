package settings

import "time"

// TenantSettings holds tenant-wide localization defaults. A tenant with no
// row yet is represented by a zero-value TenantSettings (empty strings) —
// there is no ErrNotFound; Get always succeeds.
type TenantSettings struct {
	TenantID        string
	DefaultCurrency string
	Timezone        string
	DateFormat      string
	Language        string
	UpdatedAt       time.Time
}
