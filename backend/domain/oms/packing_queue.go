package oms

import "time"

// QueueLockActivity names which session holds an order in the packing
// queue.
type QueueLockActivity string

const (
	QueueLockPicking QueueLockActivity = "picking"
	QueueLockPacking QueueLockActivity = "packing"
)

// QueueLock is who is actively working an order and since when -- the
// packing queue shows such orders read-only to everyone else.
type QueueLock struct {
	Activity QueueLockActivity
	UserID   string
	Since    time.Time
}

// PackingQueueEntry is one order in the packing queue, with its active
// lock (nil when nobody holds it, or the holder went stale).
type PackingQueueEntry struct {
	Order SalesOrder
	Lock  *QueueLock
}

// NewPackingQueueEntry derives an order's queue lock from its open picking
// and packing sessions (either may be nil). An active packing lock wins --
// packing is the later stage, and starting it closes the picking session.
func NewPackingQueueEntry(order SalesOrder, picking *PickingSession, packing *PackingSession, now time.Time) PackingQueueEntry {
	entry := PackingQueueEntry{Order: order}
	if packing != nil {
		entry.Lock = queueLock(QueueLockPacking, packing.Lock(), now)
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
