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
// A value that fails ValidateTimezone (saved before validation existed)
// also falls back to UTC, so an invalid zone can never reach SQL.
func (s TenantSettings) ReportingTimezone() string {
	if s.Timezone == "" || ValidateTimezone(s.Timezone) != nil {
		return defaultReportingTimezone
	}
	return s.Timezone
}

// ReportingLocation is ReportingTimezone as a *time.Location, for
// computing "start of this month/day" boundaries in Go.
func (s TenantSettings) ReportingLocation() *time.Location {
	loc, err := time.LoadLocation(s.ReportingTimezone())
	if err != nil {
		// Unreachable: ReportingTimezone only returns validated zones.
		return time.UTC
	}
	return loc
}

// Today is now's calendar date in the reporting timezone, as UTC midnight
// -- the form calendar-date columns (DATE) are compared against. At 00:30
// in Amsterdam it's already the new day, even though UTC is still on the
// previous one.
func (s TenantSettings) Today(now time.Time) time.Time {
	local := now.In(s.ReportingLocation())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}
