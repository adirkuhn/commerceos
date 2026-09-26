package settings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func TestParseDateFormat(t *testing.T) {
	for _, s := range []string{"DD/MM/YYYY", "MM/DD/YYYY", "YYYY-MM-DD"} {
		f, err := settings.ParseDateFormat(s)
		require.NoError(t, err)
		assert.Equal(t, settings.DateFormat(s), f)
	}

	f, err := settings.ParseDateFormat("")
	require.NoError(t, err)
	assert.Equal(t, settings.DateFormatDayMonthYear, f, "empty means the default")

	_, err = settings.ParseDateFormat("D.M.YY")
	require.ErrorIs(t, err, settings.ErrInvalidDateFormat)
}

func TestDateFormat_Format(t *testing.T) {
	day := time.Date(2027, 11, 26, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "26/11/2027", settings.DateFormatDayMonthYear.Format(day))
	assert.Equal(t, "11/26/2027", settings.DateFormatMonthDayYear.Format(day))
	assert.Equal(t, "2027-11-26", settings.DateFormatISO.Format(day))
	assert.Equal(t, "26/11/2027", settings.DateFormat("").Format(day), "unset formats as the default")
}
