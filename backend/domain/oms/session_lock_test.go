package oms_test

import (
	"testing"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
)

func lockHeldBy(user string, heartbeat time.Time) omsDomain.SessionLock {
	since := heartbeat.Add(-10 * time.Minute)
	return omsDomain.SessionLock{HolderID: &user, HeartbeatAt: &heartbeat, Since: &since}
}

func TestSessionLock_TTLIsTwoMinutes(t *testing.T) {
	assert.Equal(t, 2*time.Minute, omsDomain.SessionLockTTL)
}

func TestSessionLock_Holder(t *testing.T) {
	now := time.Now()

	holder, held := lockHeldBy("u1", now.Add(-30*time.Second)).Holder(now)
	assert.True(t, held)
	assert.Equal(t, "u1", holder)

	_, held = lockHeldBy("u1", now.Add(-omsDomain.SessionLockTTL-time.Second)).Holder(now)
	assert.False(t, held, "a lock without a heartbeat for longer than the TTL is stale")

	_, held = omsDomain.SessionLock{}.Holder(now)
	assert.False(t, held, "no holder means unlocked")
}

func TestSessionLock_IsHeldByOther(t *testing.T) {
	now := time.Now()
	fresh := lockHeldBy("u1", now)

	assert.False(t, fresh.IsHeldByOther("u1", now), "own lock")
	assert.True(t, fresh.IsHeldByOther("u2", now))
	assert.False(t, lockHeldBy("u1", now.Add(-time.Hour)).IsHeldByOther("u2", now), "stale lock holds nobody out")
}

func TestSessionLock_CanTakeOver(t *testing.T) {
	now := time.Now()
	fresh := lockHeldBy("u1", now)

	assert.True(t, fresh.CanTakeOver(omsDomain.LockActor{UserID: "u1"}, now), "holder keeps it")
	assert.False(t, fresh.CanTakeOver(omsDomain.LockActor{UserID: "u2"}, now), "fresh lock blocks a regular user")
	assert.True(t, fresh.CanTakeOver(omsDomain.LockActor{UserID: "u2", IsAdmin: true}, now), "admins may always take over")
	assert.True(t, lockHeldBy("u1", now.Add(-time.Hour)).CanTakeOver(omsDomain.LockActor{UserID: "u2"}, now), "stale lock is free")
}

func TestSessions_ExposeTheirLock(t *testing.T) {
	now := time.Now()
	user := "u1"
	pick := omsDomain.PickingSession{LockedBy: &user, LockedAt: &now, LockedSince: &now}
	pack := omsDomain.PackingSession{LockedBy: &user, LockedAt: &now, LockedSince: &now}

	assert.Equal(t, omsDomain.SessionLock{HolderID: &user, HeartbeatAt: &now, Since: &now}, pick.Lock())
	assert.Equal(t, omsDomain.SessionLock{HolderID: &user, HeartbeatAt: &now, Since: &now}, pack.Lock())
}

func TestSessionLock_AcquiredBy(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)
	own := lockHeldBy("u1", earlier)

	kept := own.AcquiredBy("u1", now)
	assert.Equal(t, "u1", *kept.HolderID)
	assert.Equal(t, now, *kept.HeartbeatAt)
	assert.Equal(t, *own.Since, *kept.Since, "same holder keeps its since")

	taken := own.AcquiredBy("u2", now)
	assert.Equal(t, "u2", *taken.HolderID)
	assert.Equal(t, now, *taken.Since, "new holder starts now")

	fresh := omsDomain.SessionLock{}.AcquiredBy("u1", now)
	assert.Equal(t, now, *fresh.Since)
}
