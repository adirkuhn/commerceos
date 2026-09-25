package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func TestValidateTimezone(t *testing.T) {
	assert.NoError(t, settings.ValidateTimezone("Europe/Amsterdam"))
	assert.NoError(t, settings.ValidateTimezone("UTC"))
	assert.NoError(t, settings.ValidateTimezone(""), "unset is allowed -- reports then use UTC")
	assert.ErrorIs(t, settings.ValidateTimezone("Mars/Olympus"), settings.ErrInvalidTimezone)
	assert.ErrorIs(t, settings.ValidateTimezone("Local"), settings.ErrInvalidTimezone, "Local is the server's zone, not a tenant's")
}

func TestTenantSettings_ReportingTimezone(t *testing.T) {
	assert.Equal(t, "Europe/Amsterdam", settings.TenantSettings{Timezone: "Europe/Amsterdam"}.ReportingTimezone())
	assert.Equal(t, "UTC", settings.TenantSettings{}.ReportingTimezone(), "unset falls back to UTC")
	assert.Equal(t, "UTC", settings.TenantSettings{Timezone: "Mars/Olympus"}.ReportingTimezone(),
		"a value saved before validation existed must not reach SQL")
}

func TestTenantSettings_ReportingLocation(t *testing.T) {
	loc := settings.TenantSettings{Timezone: "Europe/Amsterdam"}.ReportingLocation()
	assert.Equal(t, "Europe/Amsterdam", loc.String())
	assert.Equal(t, "UTC", settings.TenantSettings{}.ReportingLocation().String())
}
