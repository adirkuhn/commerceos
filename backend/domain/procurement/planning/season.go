package planning

import (
	"math"
	"time"
)

// Window is a half-open range of calendar days [Start, End).
type Window struct {
	Start time.Time
	End   time.Time
}

func (w Window) Days() int {
	return int(math.Round(w.End.Sub(w.Start).Hours() / 24))
}

// ShiftYears moves the window by whole calendar years -- "the same dates
// last year" for a custom season.
func (w Window) ShiftYears(years int) Window {
	return Window{Start: w.Start.AddDate(years, 0, 0), End: w.End.AddDate(years, 0, 0)}
}

type SeasonEventKey string

const (
	SeasonChristmas    SeasonEventKey = "christmas"
	SeasonEaster       SeasonEventKey = "easter"
	SeasonCarnival     SeasonEventKey = "carnival"
	SeasonSinterklaas  SeasonEventKey = "sinterklaas"
	SeasonValentines   SeasonEventKey = "valentines"
	SeasonMothersDay   SeasonEventKey = "mothers_day"
	SeasonFathersDayBR SeasonEventKey = "fathers_day_br"
	SeasonFestaJunina  SeasonEventKey = "festa_junina"
	SeasonBlackFriday  SeasonEventKey = "black_friday"
	// Dia dos Namorados is Brazil's Valentine's Day.
	SeasonDiaDosNamorados SeasonEventKey = "dia_dos_namorados"
	SeasonFathersDayNL    SeasonEventKey = "fathers_day_nl"
	SeasonKingsDay        SeasonEventKey = "kings_day"
	SeasonHalloween       SeasonEventKey = "halloween"
	SeasonDiaDasCriancas  SeasonEventKey = "dia_das_criancas"
	SeasonSintMaarten     SeasonEventKey = "sint_maarten"
	SeasonNewYearsEve     SeasonEventKey = "new_years_eve"
)

// anchorRule finds a holiday's date in a given year.
type anchorRule interface {
	date(year int) time.Time
}

type fixedDate struct {
	month time.Month
	day   int
}

func (f fixedDate) date(year int) time.Time { return utcDay(year, f.month, f.day) }

// notOnSunday is a fixed date moved one day earlier when it falls on a
// Sunday (Koningsdag: April 27, or Saturday the 26th).
type notOnSunday struct{ fixedDate }

func (n notOnSunday) date(year int) time.Time {
	d := n.fixedDate.date(year)
	if d.Weekday() == time.Sunday {
		return d.AddDate(0, 0, -1)
	}
	return d
}

// easterOffset is a holiday that moves with Easter (Carnival = Easter - 47).
type easterOffset struct{ days int }

func (e easterOffset) date(year int) time.Time { return EasterSunday(year).AddDate(0, 0, e.days) }

// nthWeekday is e.g. "second Sunday of May", plus offsetDays (Black Friday
// is the day after the fourth Thursday of November).
type nthWeekday struct {
	n          int
	weekday    time.Weekday
	month      time.Month
	offsetDays int
}

func (r nthWeekday) date(year int) time.Time {
	first := utcDay(year, r.month, 1)
	shift := (int(r.weekday) - int(first.Weekday()) + 7) % 7
	return first.AddDate(0, 0, shift+7*(r.n-1)+r.offsetDays)
}

// SeasonEvent is a recurring selling season around a holiday: the window
// runs DaysBefore days ahead of the holiday through DaysAfter days after it.
type SeasonEvent struct {
	Key        SeasonEventKey
	DaysBefore int
	DaysAfter  int
	anchor     anchorRule
}

func (e SeasonEvent) Anchor(year int) time.Time { return e.anchor.date(year) }

// Window is the event's selling season in a year. Last year's season of
// a moving holiday (Easter) is Window(year-1) -- aligned on the holiday,
// not on the calendar date.
func (e SeasonEvent) Window(year int) Window {
	a := e.Anchor(year)
	return Window{Start: a.AddDate(0, 0, -e.DaysBefore), End: a.AddDate(0, 0, e.DaysAfter+1)}
}

var seasonEvents = []SeasonEvent{
	{Key: SeasonChristmas, DaysBefore: 42, DaysAfter: 1, anchor: fixedDate{time.December, 25}},
	{Key: SeasonSinterklaas, DaysBefore: 28, DaysAfter: 0, anchor: fixedDate{time.December, 5}},
	{Key: SeasonBlackFriday, DaysBefore: 7, DaysAfter: 3, anchor: nthWeekday{4, time.Thursday, time.November, 1}},
	{Key: SeasonEaster, DaysBefore: 42, DaysAfter: 1, anchor: easterOffset{0}},
	{Key: SeasonCarnival, DaysBefore: 21, DaysAfter: 0, anchor: easterOffset{-47}},
	{Key: SeasonValentines, DaysBefore: 14, DaysAfter: 0, anchor: fixedDate{time.February, 14}},
	{Key: SeasonMothersDay, DaysBefore: 14, DaysAfter: 0, anchor: nthWeekday{2, time.Sunday, time.May, 0}},
	{Key: SeasonFathersDayBR, DaysBefore: 14, DaysAfter: 0, anchor: nthWeekday{2, time.Sunday, time.August, 0}},
	{Key: SeasonFestaJunina, DaysBefore: 23, DaysAfter: 6, anchor: fixedDate{time.June, 24}},
	{Key: SeasonDiaDosNamorados, DaysBefore: 14, DaysAfter: 0, anchor: fixedDate{time.June, 12}},
	{Key: SeasonFathersDayNL, DaysBefore: 14, DaysAfter: 0, anchor: nthWeekday{3, time.Sunday, time.June, 0}},
	{Key: SeasonKingsDay, DaysBefore: 14, DaysAfter: 0, anchor: notOnSunday{fixedDate{time.April, 27}}},
	{Key: SeasonHalloween, DaysBefore: 21, DaysAfter: 0, anchor: fixedDate{time.October, 31}},
	{Key: SeasonDiaDasCriancas, DaysBefore: 14, DaysAfter: 0, anchor: fixedDate{time.October, 12}},
	{Key: SeasonSintMaarten, DaysBefore: 14, DaysAfter: 0, anchor: fixedDate{time.November, 11}},
	{Key: SeasonNewYearsEve, DaysBefore: 7, DaysAfter: 0, anchor: fixedDate{time.December, 31}},
}

// SeasonEvents lists the built-in events.
func SeasonEvents() []SeasonEvent {
	return append([]SeasonEvent(nil), seasonEvents...)
}

func FindSeasonEvent(key SeasonEventKey) (SeasonEvent, bool) {
	for _, e := range seasonEvents {
		if e.Key == key {
			return e, true
		}
	}
	return SeasonEvent{}, false
}

// EasterSunday is the Gregorian Easter date (anonymous Gregorian algorithm).
func EasterSunday(year int) time.Time {
	a := year % 19
	b, c := year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	dayOfMonth := (h+l-7*m+114)%31 + 1
	return utcDay(year, time.Month(month), dayOfMonth)
}

func utcDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
