package oms_test

import (
	"testing"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueEntry(t *testing.T) {
	now := time.Now()
	since := now.Add(-12 * time.Minute)
	fresh := now.Add(-30 * time.Second)
	stale := now.Add(-time.Hour)
	picker, packer := "picker", "packer"
	order := omsDomain.SalesOrder{ID: "so-1"}

	t.Run("no sessions -> unlocked", func(t *testing.T) {
		e := omsDomain.NewQueueEntry(order, nil, nil, false, now)
		assert.Equal(t, "so-1", e.Order.ID)
		assert.Nil(t, e.Lock)
	})

	t.Run("active packing lock wins", func(t *testing.T) {
		pick := &omsDomain.PickingSession{LockedBy: &picker, LockedAt: &fresh, LockedSince: &since}
		pack := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &fresh, LockedSince: &since}
		e := omsDomain.NewQueueEntry(order, pick, pack, false, now)
		require.NotNil(t, e.Lock)
		assert.Equal(t, omsDomain.QueueLockPacking, e.Lock.Activity)
		assert.Equal(t, "packer", e.Lock.UserID)
		assert.Equal(t, since, e.Lock.Since)
	})

	t.Run("active picking lock", func(t *testing.T) {
		pick := &omsDomain.PickingSession{LockedBy: &picker, LockedAt: &fresh, LockedSince: &since}
		e := omsDomain.NewQueueEntry(order, pick, nil, false, now)
		require.NotNil(t, e.Lock)
		assert.Equal(t, omsDomain.QueueLockPicking, e.Lock.Activity)
		assert.Equal(t, "picker", e.Lock.UserID)
	})

	t.Run("stale locks are free", func(t *testing.T) {
		pick := &omsDomain.PickingSession{LockedBy: &picker, LockedAt: &stale, LockedSince: &since}
		pack := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &stale, LockedSince: &since}
		assert.Nil(t, omsDomain.NewQueueEntry(order, pick, pack, false, now).Lock)
	})

	t.Run("lock without since falls back to heartbeat", func(t *testing.T) {
		pack := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &fresh}
		e := omsDomain.NewQueueEntry(order, nil, pack, false, now)
		require.NotNil(t, e.Lock)
		assert.Equal(t, fresh, e.Lock.Since)
	})
}

func TestNewQueueEntry_InPacking(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-30 * time.Second)
	stale := now.Add(-time.Hour)
	packer := "packer"
	order := omsDomain.SalesOrder{ID: "so-1"}

	assert.False(t, omsDomain.NewQueueEntry(order, nil, nil, false, now).InPacking)

	opened := &omsDomain.PackingSession{}
	assert.False(t, omsDomain.NewQueueEntry(order, nil, opened, false, now).InPacking, "a packing session merely opened doesn't block picking")

	active := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &fresh}
	assert.True(t, omsDomain.NewQueueEntry(order, nil, active, false, now).InPacking, "someone is packing it right now")

	left := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &stale}
	assert.True(t, omsDomain.NewQueueEntry(order, nil, left, true, now).InPacking, "packing already recorded something")
}

func TestPackingSession_BlocksPicking(t *testing.T) {
	now := time.Now()
	fresh, stale := now.Add(-30*time.Second), now.Add(-time.Hour)
	packer := "packer"

	assert.False(t, omsDomain.PackingSession{}.BlocksPicking(false, now))
	assert.True(t, omsDomain.PackingSession{LockedBy: &packer, LockedAt: &fresh}.BlocksPicking(false, now))
	assert.False(t, omsDomain.PackingSession{LockedBy: &packer, LockedAt: &stale}.BlocksPicking(false, now))
	assert.True(t, omsDomain.PackingSession{}.BlocksPicking(true, now))
}
