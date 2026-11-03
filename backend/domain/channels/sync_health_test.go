package channels

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestApplyRunOutcome(t *testing.T) {
	t0 := time.Date(2026, 10, 31, 9, 23, 0, 0, time.UTC)
	at := func(d time.Duration) time.Time { return t0.Add(d) }
	ptr := func(tm time.Time) *time.Time { return &tm }

	cases := []struct {
		name         string
		failingSince *time.Time
		alerted      bool
		ranAt        time.Time
		failed       bool
		want         SyncHealthChange
		wantSince    *time.Time
		wantAlerted  bool
	}{
		{"healthy run stays quiet", nil, false, at(0), false, SyncHealthUnchanged, nil, false},
		{"first failure starts the clock, no alert yet", nil, false, at(0), true, SyncHealthUnchanged, ptr(at(0)), false},
		{"still failing under the threshold", ptr(at(0)), false, at(59 * time.Minute), true, SyncHealthUnchanged, ptr(at(0)), false},
		{"failing for the threshold alerts once", ptr(at(0)), false, at(SyncFailureAlertAfter), true, SyncHealthFailing, ptr(at(0)), true},
		{"already alerted stays quiet", ptr(at(0)), true, at(3 * 24 * time.Hour), true, SyncHealthUnchanged, ptr(at(0)), true},
		{"success after an alert recovers", ptr(at(0)), true, at(3 * 24 * time.Hour), false, SyncHealthRecovered, nil, false},
		{"success after a short blip is quiet", ptr(at(0)), false, at(10 * time.Minute), false, SyncHealthUnchanged, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := SyncSchedule{FailingSince: tc.failingSince, FailureAlerted: tc.alerted}

			got := s.ApplyRunOutcome(tc.ranAt, tc.failed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantSince, s.FailingSince)
			assert.Equal(t, tc.wantAlerted, s.FailureAlerted)
		})
	}
}
