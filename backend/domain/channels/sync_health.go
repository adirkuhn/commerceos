package channels

import "time"

// SyncFailureAlertAfter is how long a schedule must keep failing before
// staff are alerted -- long enough to ride out a timeout or a store
// restart, short enough that a dead store is noticed the same morning
// (2026-10-31..11-03 a Woo 403 went unnoticed for three days).
const SyncFailureAlertAfter = time.Hour

// SyncHealthChange is what one run's outcome means for alerting.
type SyncHealthChange int

const (
	SyncHealthUnchanged SyncHealthChange = iota
	// SyncHealthFailing: the schedule just crossed SyncFailureAlertAfter.
	SyncHealthFailing
	// SyncHealthRecovered: the first success after an alert.
	SyncHealthRecovered
)

// ApplyRunOutcome folds one run's result into the schedule's failure
// streak and says whether staff must hear about it: once when a streak
// reaches SyncFailureAlertAfter, once again when it ends. A blip that
// recovers before the threshold is never reported.
func (s *SyncSchedule) ApplyRunOutcome(ranAt time.Time, failed bool) SyncHealthChange {
	if !failed {
		wasAlerted := s.FailureAlerted
		s.FailingSince, s.FailureAlerted = nil, false
		if wasAlerted {
			return SyncHealthRecovered
		}
		return SyncHealthUnchanged
	}
	if s.FailingSince == nil {
		since := ranAt
		s.FailingSince = &since
	}
	if !s.FailureAlerted && ranAt.Sub(*s.FailingSince) >= SyncFailureAlertAfter {
		s.FailureAlerted = true
		return SyncHealthFailing
	}
	return SyncHealthUnchanged
}
