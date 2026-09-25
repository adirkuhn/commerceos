package settings

import (
	"strings"
	"time"
)

// defaultReportingTimezone is what reports bucket by for a tenant that
// never set a timezone -- the behaviour before tenants could choose one.
const defaultReportingTimezone = "UTC"

// ValidateTimezone accepts an IANA zone name (e.g. "Europe/Amsterdam") or
// empty (unset). The value ends up in SQL `AT TIME ZONE` / date_trunc, so
// anything the tz database doesn't know must be rejected here, not fail
// every report later. "Local" is the server's own zone, never a tenant's.
func ValidateTimezone(tz string) error {
	if tz == "" {
		return nil
	}
	if strings.EqualFold(tz, "Local") {
		return ErrInvalidTimezone
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ErrInvalidTimezone
	}
	return nil
}

// ReportingTimezone is the zone reports and the dashboard use for "day",
// "week", "month" and weekday boundaries -- the tenant's own, so an order
// at 00:30 local time lands on that local day, not the previous UTC one.
func (s TenantSettings) ReportingTimezone() string {
	if s.Timezone == "" {
		return defaultReportingTimezone
	}
	return s.Timezone
}
