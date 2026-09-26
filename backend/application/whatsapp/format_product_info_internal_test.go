package whatsapp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

func TestFormatProductInfo_expiryFollowsTenantDateFormat(t *testing.T) {
	expiry := time.Date(2027, 11, 26, 0, 0, 0, 0, time.UTC)
	info := ProductInfo{NearestExpiry: &expiry}

	info.DateFormat = settingsDomain.DateFormatDayMonthYear
	assert.Equal(t, "Nearest expiry: 26/11/2027", formatProductInfo("EXPIRY", info))

	info.DateFormat = settingsDomain.DateFormatMonthDayYear
	assert.Equal(t, "Nearest expiry: 11/26/2027", formatProductInfo("EXPIRY", info))
}
