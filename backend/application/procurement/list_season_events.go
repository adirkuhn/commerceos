package procurement

import (
	"sort"
	"time"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

// SeasonEventOccurrence is a built-in season event in a given year.
type SeasonEventOccurrence struct {
	Key planning.SeasonEventKey
	// Year/Date: the holiday itself; Window: its selling season.
	Year   int
	Date   time.Time
	Window planning.Window
}

// ListSeasonEventsUseCase lists the built-in season events, holiday soonest
// first -- for picking which season to plan.
type ListSeasonEventsUseCase struct {
	now func() time.Time
}

func NewListSeasonEventsUseCase(now func() time.Time) *ListSeasonEventsUseCase {
	return &ListSeasonEventsUseCase{now: now}
}

// Execute lists each event in year, or its next occurrence (season not
// over yet) when year is 0.
func (uc *ListSeasonEventsUseCase) Execute(year int) []SeasonEventOccurrence {
	asOf := startOfDay(uc.now().UTC())
	events := planning.SeasonEvents()
	out := make([]SeasonEventOccurrence, len(events))
	for i, e := range events {
		y := year
		if y == 0 {
			y = nextOccurrenceYear(e, asOf)
		}
		out[i] = SeasonEventOccurrence{Key: e.Key, Year: y, Date: e.Anchor(y), Window: e.Window(y)}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Date.Before(out[b].Date) })
	return out
}
