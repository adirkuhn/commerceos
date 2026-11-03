package channels_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

type capturingNotifier struct {
	got []appnotifications.CreateNotificationInput
	err error
}

func (c *capturingNotifier) Execute(_ context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error) {
	c.got = append(c.got, in)
	return &notificationsDomain.Notification{}, c.err
}

type subjectsOutbox struct{ subjects []string }

func (o *subjectsOutbox) Enqueue(_ context.Context, subject, _ string, payload []byte, _ time.Time) error {
	o.subjects = append(o.subjects, subject)
	var probe map[string]any
	return json.Unmarshal(payload, &probe)
}

type recordSyncRunHarness struct {
	schedules *stubSyncScheduleRepo
	outbox    *subjectsOutbox
	notifier  *capturingNotifier
	uc        *appchannels.RecordSyncRunUseCase
}

func newRecordSyncRunHarness(t *testing.T) *recordSyncRunHarness {
	h := &recordSyncRunHarness{
		schedules: &stubSyncScheduleRepo{}, outbox: &subjectsOutbox{}, notifier: &capturingNotifier{},
	}
	conns := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", StoreURL: "https://cantinho.example/"}}
	h.uc = appchannels.NewRecordSyncRunUseCase(newMockGormDB(t, 1), h.schedules, conns, h.outbox, h.notifier)
	return h
}

var syncRanAt = time.Date(2026, 11, 3, 7, 0, 0, 0, time.UTC)

func ordersSchedule() channelsDomain.SyncSchedule {
	return channelsDomain.SyncSchedule{
		ID: "sched-1", TenantID: "tenant-1", ConnectionID: "conn-1", IntervalMinutes: 15,
		EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
	}
}

func TestRecordSyncRun_successRecordsTheRunAndStaysQuiet(t *testing.T) {
	h := newRecordSyncRunHarness(t)

	err := h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{Schedule: ordersSchedule(), RanAt: syncRanAt})

	require.NoError(t, err)
	require.Len(t, h.schedules.recorded, 1)
	run := h.schedules.recorded[0]
	assert.Equal(t, "success", run.Status)
	assert.True(t, run.AdvanceLastRunAt)
	assert.Equal(t, syncRanAt, run.CursorAt, "no cursor of its own -> wall clock")
	assert.Equal(t, syncRanAt.Add(15*time.Minute), run.NextRunAt)
	assert.Empty(t, h.schedules.health, "health unchanged -> no write")
	assert.Empty(t, h.outbox.subjects)
	assert.Empty(t, h.notifier.got)
}

func TestRecordSyncRun_aFirstFailureStartsTheStreakWithoutAlerting(t *testing.T) {
	h := newRecordSyncRunHarness(t)

	err := h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{
		Schedule: ordersSchedule(), RanAt: syncRanAt, RunErr: errors.New("403"),
	})

	require.NoError(t, err)
	assert.Equal(t, "error", h.schedules.recorded[0].Status)
	assert.Equal(t, "403", h.schedules.recorded[0].ErrMsg)
	assert.False(t, h.schedules.recorded[0].AdvanceLastRunAt)
	require.Len(t, h.schedules.health, 1)
	assert.Equal(t, syncRanAt, *h.schedules.health[0].FailingSince)
	assert.False(t, h.schedules.health[0].FailureAlerted)
	assert.Empty(t, h.notifier.got)
}

func TestRecordSyncRun_failingForAnHourAlertsWithAnEventAndANotification(t *testing.T) {
	h := newRecordSyncRunHarness(t)
	s := ordersSchedule()
	since := syncRanAt.Add(-channelsDomain.SyncFailureAlertAfter)
	s.FailingSince = &since

	err := h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{
		Schedule: s, RanAt: syncRanAt, RunErr: errors.New("woocommerce request failed: " + strings.Repeat("x", 1000)),
	})

	require.NoError(t, err)
	require.Len(t, h.schedules.health, 1)
	assert.True(t, h.schedules.health[0].FailureAlerted)
	assert.Equal(t, []string{events.SyncScheduleFailureAlerted.Subject}, h.outbox.subjects)
	require.Len(t, h.notifier.got, 1)
	n := h.notifier.got[0]
	assert.Equal(t, notificationsDomain.TypeChannelSyncFailing, n.Type)
	assert.Equal(t, "cantinho.example", n.Params["store"])
	assert.LessOrEqual(t, len(n.Params["error"].(string)), 301+len("…"), "a long error is cut short")
	assert.Equal(t, "/integrations", n.Link)
}

func TestRecordSyncRun_successAfterAnAlertRecovers(t *testing.T) {
	h := newRecordSyncRunHarness(t)
	s := ordersSchedule()
	since := syncRanAt.Add(-72 * time.Hour)
	s.FailingSince, s.FailureAlerted = &since, true

	err := h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{Schedule: s, RanAt: syncRanAt})

	require.NoError(t, err)
	require.Len(t, h.schedules.health, 1)
	assert.Nil(t, h.schedules.health[0].FailingSince)
	assert.Equal(t, []string{events.SyncScheduleRecovered.Subject}, h.outbox.subjects)
	require.Len(t, h.notifier.got, 1)
	assert.Equal(t, notificationsDomain.TypeChannelSyncRecovered, h.notifier.got[0].Type)
}

func TestRecordSyncRun_aNotificationFailureDoesNotFailTheRecording(t *testing.T) {
	h := newRecordSyncRunHarness(t)
	h.notifier.err = errors.New("smtp down")
	s := ordersSchedule()
	since := syncRanAt.Add(-2 * time.Hour)
	s.FailingSince = &since

	err := h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{Schedule: s, RanAt: syncRanAt, RunErr: errors.New("403")})

	assert.NoError(t, err)
	assert.Len(t, h.notifier.got, 1)
}

func TestRecordSyncRun_usesTheRunnersOwnCursor(t *testing.T) {
	h := newRecordSyncRunHarness(t)
	cursor := syncRanAt.Add(-time.Minute)

	require.NoError(t, h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{
		Schedule: ordersSchedule(), RanAt: syncRanAt, CursorAt: &cursor,
	}))

	assert.Equal(t, cursor, h.schedules.recorded[0].CursorAt)
}

func TestRecordSyncRun_aStorageFailureIsReturned(t *testing.T) {
	h := newRecordSyncRunHarness(t)
	h.schedules.healthErr = errors.New("db down")

	err := h.uc.Execute(context.Background(), appchannels.RecordSyncRunInput{Schedule: ordersSchedule(), RanAt: syncRanAt, RunErr: errors.New("403")})

	assert.Error(t, err)
	assert.Empty(t, h.notifier.got, "nothing stored -> nobody told")
}
