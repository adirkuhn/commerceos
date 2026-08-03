package inventory

import "time"

// Warehouse address fields mirror Shopify's Location resource
// (https://shopify.dev/docs/api/admin-rest/latest/resources/location) —
// the shape merchants coming from Shopify already expect for a fulfillment
// location.
type Warehouse struct {
	ID        string
	TenantID  string
	Name      string
	Address1  string
	Address2  string
	City      string
	Province  string
	Zip       string
	Country   string
	Phone     string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
