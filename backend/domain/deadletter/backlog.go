package deadletter

import (
	"sort"
	"time"
)

// StaleAfter is how long a dead letter may wait before the daily alert
// escalates from "waiting" to "stale".
const StaleAfter = 24 * time.Hour

// Backlog is one tenant's dead letters awaiting a super-admin. TenantID is
// empty for dead letters whose data names no tenant.
type Backlog struct {
	TenantID string
	Count    int
	Oldest   time.Time
}

func (b Backlog) Stale(now time.Time) bool { return now.Sub(b.Oldest) >= StaleAfter }

// AgeHours is how many whole hours the oldest dead letter has waited.
func (b Backlog) AgeHours(now time.Time) int { return int(now.Sub(b.Oldest) / time.Hour) }

// waitingSince is when a dead letter started waiting: when it was
// dead-lettered, or, for one without that header, when the stream stored it.
func (d DeadLetter) waitingSince() time.Time {
	if d.DeadLetteredAt != nil {
		return *d.DeadLetteredAt
	}
	return d.StoredAt
}

// Summarize groups dead letters by tenant, sorted by tenant ID.
func Summarize(dls []DeadLetter) []Backlog {
	byTenant := map[string]*Backlog{}
	for _, d := range dls {
		b, ok := byTenant[d.TenantID]
		if !ok {
			b = &Backlog{TenantID: d.TenantID, Oldest: d.waitingSince()}
			byTenant[d.TenantID] = b
		}
		b.Count++
		if since := d.waitingSince(); since.Before(b.Oldest) {
			b.Oldest = since
		}
	}
	out := make([]Backlog, 0, len(byTenant))
	for _, b := range byTenant {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TenantID < out[j].TenantID })
	return out
}
