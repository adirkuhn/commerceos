package oms_test

import (
	"testing"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackingQueueEntry(t *testing.T) {
	now := time.Now()
	since := now.Add(-12 * time.Minute)
	fresh := now.Add(-30 * time.Second)
	stale := now.Add(-time.Hour)
	picker, packer := "picker", "packer"
	order := omsDomain.SalesOrder{ID: "so-1"}

	t.Run("no sessions -> unlocked", func(t *testing.T) {
		e := omsDomain.NewPackingQueueEntry(order, nil, nil, now)
		assert.Equal(t, "so-1", e.Order.ID)
		assert.Nil(t, e.Lock)
	})

	t.Run("active packing lock wins", func(t *testing.T) {
		pick := &omsDomain.PickingSession{LockedBy: &picker, LockedAt: &fresh, LockedSince: &since}
		pack := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &fresh, LockedSince: &since}
		e := omsDomain.NewPackingQueueEntry(order, pick, pack, now)
		require.NotNil(t, e.Lock)
		assert.Equal(t, omsDomain.QueueLockPacking, e.Lock.Activity)
		assert.Equal(t, "packer", e.Lock.UserID)
		assert.Equal(t, since, e.Lock.Since)
	})

	t.Run("active picking lock", func(t *testing.T) {
		pick := &omsDomain.PickingSession{LockedBy: &picker, LockedAt: &fresh, LockedSince: &since}
		e := omsDomain.NewPackingQueueEntry(order, pick, nil, now)
		require.NotNil(t, e.Lock)
		assert.Equal(t, omsDomain.QueueLockPicking, e.Lock.Activity)
		assert.Equal(t, "picker", e.Lock.UserID)
	})

	t.Run("stale locks are free", func(t *testing.T) {
		pick := &omsDomain.PickingSession{LockedBy: &picker, LockedAt: &stale, LockedSince: &since}
		pack := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &stale, LockedSince: &since}
		assert.Nil(t, omsDomain.NewPackingQueueEntry(order, pick, pack, now).Lock)
	})

	t.Run("lock without since falls back to heartbeat", func(t *testing.T) {
		pack := &omsDomain.PackingSession{LockedBy: &packer, LockedAt: &fresh}
		e := omsDomain.NewPackingQueueEntry(order, nil, pack, now)
		require.NotNil(t, e.Lock)
		assert.Equal(t, fresh, e.Lock.Since)
	})
}
