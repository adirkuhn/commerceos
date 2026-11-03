package deadletter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/deadletter"
)

func TestSummarize_groupsByTenantWithOldest(t *testing.T) {
	t0 := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	earlier := t0.Add(-3 * time.Hour)
	dls := []deadletter.DeadLetter{
		{TenantID: "t-2", StoredAt: t0},
		{TenantID: "t-1", StoredAt: t0, DeadLetteredAt: &earlier},
		{TenantID: "t-1", StoredAt: t0.Add(-time.Hour)},
		{TenantID: "", StoredAt: t0},
	}

	got := deadletter.Summarize(dls)

	assert.Equal(t, []deadletter.Backlog{
		{TenantID: "", Count: 1, Oldest: t0},
		{TenantID: "t-1", Count: 2, Oldest: earlier},
		{TenantID: "t-2", Count: 1, Oldest: t0},
	}, got)
}

func TestSummarize_empty(t *testing.T) {
	assert.Empty(t, deadletter.Summarize(nil))
}

func TestBacklog_Stale(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

	assert.False(t, deadletter.Backlog{Oldest: now.Add(-deadletter.StaleAfter + time.Minute)}.Stale(now))
	assert.True(t, deadletter.Backlog{Oldest: now.Add(-deadletter.StaleAfter)}.Stale(now))
}

func TestBacklog_AgeHours(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, 30, deadletter.Backlog{Oldest: now.Add(-30*time.Hour - 20*time.Minute)}.AgeHours(now))
}
