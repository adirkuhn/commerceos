package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func TestTenantSettings_Currency(t *testing.T) {
	assert.Equal(t, "EUR", settings.TenantSettings{DefaultCurrency: "EUR"}.Currency())
	assert.Equal(t, settings.FallbackCurrency, settings.TenantSettings{}.Currency())
}
