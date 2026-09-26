package settings

import "time"

// DateFormat is how dates are shown to people for a tenant (Settings →
// Localization): in the admin app, the scanner, and text the backend
// writes for humans (e.g. a WhatsApp reply). Machine formats sent to
// other systems (WooCommerce meta, MyParcel) stay fixed and never use it.
type DateFormat string

const (
	DateFormatDayMonthYear DateFormat = "DD/MM/YYYY"
	DateFormatMonthDayYear DateFormat = "MM/DD/YYYY"
	DateFormatISO          DateFormat = "YYYY-MM-DD"
)

// ParseDateFormat validates a date format; empty means the default,
// DD/MM/YYYY (what a never-saved tenant reads as).
func ParseDateFormat(s string) (DateFormat, error) {
	switch DateFormat(s) {
	case "", DateFormatDayMonthYear:
		return DateFormatDayMonthYear, nil
	case DateFormatMonthDayYear:
		return DateFormatMonthDayYear, nil
	case DateFormatISO:
		return DateFormatISO, nil
	default:
		return "", ErrInvalidDateFormat
	}
}

// Format renders t's calendar date (in t's own location) in this format.
func (f DateFormat) Format(t time.Time) string {
	switch f {
	case DateFormatMonthDayYear:
		return t.Format("01/02/2006")
	case DateFormatISO:
		return t.Format("2006-01-02")
	default:
		return t.Format("02/01/2006")
	}
}
