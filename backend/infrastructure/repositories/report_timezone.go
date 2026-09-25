package repositories

import (
	"fmt"

	"gorm.io/gorm"
)

// useReportTimezone makes the rest of the current transaction compute
// calendar values -- date_trunc, EXTRACT(DOW), ::date, now()::date, cohort
// months -- in the tenant's zone instead of the session default (UTC), so a
// day/week/month/weekday bucket is the tenant's local one. is_local=true
// scopes it to this transaction (tenant.WithTenant's), like SET LOCAL.
// Stored and returned timestamps are timestamptz, i.e. absolute instants,
// so only calendar bucketing changes. tz must be a validated zone (see
// settings.TenantSettings.ReportingTimezone); it is bound, never inlined.
func useReportTimezone(tx *gorm.DB, tz string) error {
	if err := tx.Exec(`SELECT set_config('TimeZone', ?, true)`, tz).Error; err != nil {
		return fmt.Errorf("set report timezone %q: %w", tz, err)
	}
	return nil
}
