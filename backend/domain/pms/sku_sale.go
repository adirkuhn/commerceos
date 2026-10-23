package pms

import "time"

// ActiveSalePrice is SalePriceMinorUnits when the sale runs today (through
// SaleEndDate, inclusive), else nil -- the scheduler clears ended sales only
// once a day, so readers must not trust a stale SalePriceMinorUnits.
func (s SKU) ActiveSalePrice(today time.Time) *int64 {
	if s.SalePriceMinorUnits == nil {
		return nil
	}
	if s.SaleEndDate != nil && calendarDate(*s.SaleEndDate).Before(calendarDate(today)) {
		return nil
	}
	return s.SalePriceMinorUnits
}

func calendarDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
