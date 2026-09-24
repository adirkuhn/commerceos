package oms

import "time"

// SessionLockTTL is how long a picking/packing session lock stays valid
// without a heartbeat -- two missed beats of the frontends' 60s touch
// interval. After that anyone may take the order over.
const SessionLockTTL = 2 * time.Minute

// LockActor is who is asking for a session lock. IsAdmin callers may take a
// lock over even while its holder is still active.
type LockActor struct {
	UserID  string
	IsAdmin bool
}

// SessionLock is a picking/packing session's per-order lock: HolderID is
// the lm2 user holding it, HeartbeatAt its last touch, Since when that
// holder acquired it.
type SessionLock struct {
	HolderID    *string
	HeartbeatAt *time.Time
	Since       *time.Time
}

// Holder returns the user actively holding the lock -- false when nobody
// holds it or the holder's heartbeat is older than SessionLockTTL.
func (l SessionLock) Holder(now time.Time) (string, bool) {
	if l.HolderID == nil || l.HeartbeatAt == nil || now.Sub(*l.HeartbeatAt) >= SessionLockTTL {
		return "", false
	}
	return *l.HolderID, true
}

// IsHeldByOther reports whether someone other than userID actively holds
// the lock.
func (l SessionLock) IsHeldByOther(userID string, now time.Time) bool {
	holder, held := l.Holder(now)
	return held && holder != userID
}

// CanTakeOver reports whether actor may acquire the lock now: it's free,
// stale, already theirs, or they're an admin.
func (l SessionLock) CanTakeOver(actor LockActor, now time.Time) bool {
	return actor.IsAdmin || !l.IsHeldByOther(actor.UserID, now)
}

// AcquiredBy is the lock after userID claims or refreshes it at now --
// Since carries over only when the holder doesn't change.
func (l SessionLock) AcquiredBy(userID string, now time.Time) SessionLock {
	since := now
	if l.HolderID != nil && *l.HolderID == userID && l.Since != nil {
		since = *l.Since
	}
	return SessionLock{HolderID: &userID, HeartbeatAt: &now, Since: &since}
}

// setLock copies l back onto a session's lock fields.
func setLock(l SessionLock, holder **string, heartbeat, since **time.Time) {
	*holder, *heartbeat, *since = l.HolderID, l.HeartbeatAt, l.Since
}

// ApplyLock sets the session's lock fields from l.
func (s *PickingSession) ApplyLock(l SessionLock) {
	setLock(l, &s.LockedBy, &s.LockedAt, &s.LockedSince)
}

// ApplyLock sets the session's lock fields from l.
func (s *PackingSession) ApplyLock(l SessionLock) {
	setLock(l, &s.LockedBy, &s.LockedAt, &s.LockedSince)
}
