package oms

import "time"

// QueueLockActivity names which session holds an order in the picking or
// packing queue.
type QueueLockActivity string

const (
	QueueLockPicking QueueLockActivity = "picking"
	QueueLockPacking QueueLockActivity = "packing"
)

// QueueLock is who is actively working an order and since when -- the
// queues show such orders read-only to everyone else.
type QueueLock struct {
	Activity QueueLockActivity
	UserID   string
	Since    time.Time
}

// QueueEntry is one order in the picking or packing queue, with its active
// lock (nil when nobody holds it, or the holder went stale). InPacking is
// whether packing has the order (see PackingSession.BlocksPicking) -- the
// picking screens then show it read-only.
type QueueEntry struct {
	Order     SalesOrder
	Lock      *QueueLock
	InPacking bool
}

// NewQueueEntry derives an order's queue state from its open picking and
// packing sessions (either may be nil); packingHasScans is whether the
// packing session recorded anything yet. An active packing lock wins --
// packing is the later stage, and starting it closes the picking session.
func NewQueueEntry(order SalesOrder, picking *PickingSession, packing *PackingSession, packingHasScans bool, now time.Time) QueueEntry {
	entry := QueueEntry{Order: order}
	if packing != nil {
		entry.Lock = queueLock(QueueLockPacking, packing.Lock(), now)
		entry.InPacking = packing.BlocksPicking(packingHasScans, now)
	}
	if entry.Lock == nil && picking != nil {
		entry.Lock = queueLock(QueueLockPicking, picking.Lock(), now)
	}
	return entry
}

func queueLock(activity QueueLockActivity, l SessionLock, now time.Time) *QueueLock {
	holder, held := l.Holder(now)
	if !held {
		return nil
	}
	since := *l.HeartbeatAt
	if l.Since != nil {
		since = *l.Since
	}
	return &QueueLock{Activity: activity, UserID: holder, Since: since}
}

// BlocksPicking reports whether packing has the order: someone is actively
// packing it, or packing already recorded something (hasScans). A packing
// session merely opened to look at the order doesn't block picking.
func (s PackingSession) BlocksPicking(hasScans bool, now time.Time) bool {
	_, held := s.Lock().Holder(now)
	return held || hasScans
}
