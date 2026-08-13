// Package lm2license verifies the lm2-issued license a POS desktop
// presents on every request as its CommerceOS API credential. This is
// infrastructure (an external system's token format), not domain -- it has
// nothing to do with CommerceOS's own business rules.
package lm2license

import "time"

type DeviceBinding struct {
	DeviceID string `json:"device_id"`
	Mode     string `json:"mode"`
}

type Limits struct {
	MaxDevices   int `json:"max_devices"`
	MaxLocations int `json:"max_locations"`
	MaxUsers     int `json:"max_users"`
}

type Signing struct {
	Fingerprint string `json:"fingerprint"`
	KeyID       string `json:"key_id"`
}

type LicenseModule struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type PlanLimits struct {
	MaxIntegrations     *int `json:"max_integrations,omitempty"`
	MaxPaymentMethods   *int `json:"max_payment_methods,omitempty"`
	MaxProducts         *int `json:"max_products,omitempty"`
	MaxOrderHistory     *int `json:"max_order_history,omitempty"`
	MaxOrdersPerMonth   *int `json:"max_orders_per_month,omitempty"`
	OrdersUsedThisMonth *int `json:"orders_used_this_month,omitempty"`
}

// LicensePayload is lm2's license format, unchanged from the POS desktop's
// own verification code -- CompanyID is the field CommerceOS treats as its
// own TenantID (confirmed 1:1: both are lm2-issued identifiers for the same
// company/tenant).
type LicensePayload struct {
	CompanyID     string          `json:"company_id"`
	DeviceBinding DeviceBinding   `json:"device_binding"`
	ExpiresAt     *time.Time      `json:"expires_at"`
	IssuedAt      *time.Time      `json:"issued_at"`
	LicenseID     string          `json:"license_id"`
	Limits        Limits          `json:"limits"`
	Modules       []LicenseModule `json:"modules"`
	PlanType      string          `json:"plan_type"`
	PlanLimits    *PlanLimits     `json:"plan_limits,omitempty"`
	SchemaVersion int             `json:"schema_version"`
	Signing       Signing         `json:"signing"`
	Status        string          `json:"status"`
	UserID        string          `json:"user_id"`
	UserEmail     string          `json:"user_email"`
	UserRole      string          `json:"user_role"`
}

// IsUsable enforces expires_at/status ourselves -- TokenService.
// VerifyAndGetPayload deliberately skips the JWT wrapper's own exp/nbf/iat
// (see its doc comment: the POS desktop needs licenses to keep validating
// locally across long offline periods). That's correct for the POS's own
// "am I licensed" check but wrong for API auth, so CommerceOS enforces
// these fields itself, on the payload's own expires_at/status, every time.
func (p *LicensePayload) IsUsable(now time.Time) bool {
	if p.Status != "active" {
		return false
	}
	if p.CompanyID == "" {
		return false
	}
	if p.ExpiresAt != nil && !p.ExpiresAt.After(now) {
		return false
	}
	return true
}
