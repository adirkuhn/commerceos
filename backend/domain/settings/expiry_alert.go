package settings

import "sort"

// ExpiryAlertWindowOptions are the "days before expiry" windows a tenant can
// pick for the stock-close-to-expiry report.
var ExpiryAlertWindowOptions = []int{90, 30, 15, 7}

// ExpiryAlert is the tenant's stock-close-to-expiry report config: when
// Enabled, the scheduler reports stock expiring within each chosen window.
// Windows is sorted largest first, without duplicates; it is kept while the
// alert is disabled so re-enabling restores the merchant's choice.
type ExpiryAlert struct {
	Enabled bool
	Windows []int
}

// NewExpiryAlert validates windows against ExpiryAlertWindowOptions. An
// enabled alert needs at least one window.
func NewExpiryAlert(enabled bool, windows []int) (ExpiryAlert, error) {
	seen := make(map[int]bool, len(windows))
	out := make([]int, 0, len(windows))
	for _, w := range windows {
		if !isExpiryAlertWindowOption(w) {
			return ExpiryAlert{}, ErrInvalidExpiryAlertWindow
		}
		if seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	if enabled && len(out) == 0 {
		return ExpiryAlert{}, ErrExpiryAlertWindowsRequired
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return ExpiryAlert{Enabled: enabled, Windows: out}, nil
}

func isExpiryAlertWindowOption(w int) bool {
	for _, o := range ExpiryAlertWindowOptions {
		if o == w {
			return true
		}
	}
	return false
}

// ExpiryBand is one report: stock with MinDaysLeft..WindowDays days left
// before expiry (both inclusive). Bands never overlap, so stock expiring in
// 10 days shows up in the 15-day report only, not also in the 30 and 90 ones.
type ExpiryBand struct {
	WindowDays  int
	MinDaysLeft int
}

// Contains reports whether stock with daysLeft days before expiry belongs in
// this band.
func (b ExpiryBand) Contains(daysLeft int) bool {
	return daysLeft >= b.MinDaysLeft && daysLeft <= b.WindowDays
}

// Bands splits the chosen windows into non-overlapping reports, largest
// first. The smallest band starts at 0 days left (expires today). A disabled
// alert has no bands.
func (a ExpiryAlert) Bands() []ExpiryBand {
	if !a.Enabled {
		return nil
	}
	return bandsOf(a.Windows)
}

// ReviewBands are the bands the close-to-expiry screen shows, whether or
// not the email report is on: the chosen windows, or every option when
// none were chosen. Same bands as the report, so marking stock as seen in
// one means the same thing on screen and in the email.
func (a ExpiryAlert) ReviewBands() []ExpiryBand {
	if len(a.Windows) == 0 {
		return bandsOf(ExpiryAlertWindowOptions)
	}
	return bandsOf(a.Windows)
}

// ReviewBand is the review band of the given window, if it is one.
func (a ExpiryAlert) ReviewBand(windowDays int) (ExpiryBand, bool) {
	for _, b := range a.ReviewBands() {
		if b.WindowDays == windowDays {
			return b, true
		}
	}
	return ExpiryBand{}, false
}

// bandsOf splits windows, sorted largest first, into non-overlapping bands.
func bandsOf(windows []int) []ExpiryBand {
	bands := make([]ExpiryBand, len(windows))
	for i, w := range windows {
		minDaysLeft := 0
		if i+1 < len(windows) {
			minDaysLeft = windows[i+1] + 1
		}
		bands[i] = ExpiryBand{WindowDays: w, MinDaysLeft: minDaysLeft}
	}
	return bands
}
