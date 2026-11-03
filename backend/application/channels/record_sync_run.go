package channels

import (
	"context"
	"fmt"
	"log"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// alertErrorMaxLen keeps a run's error readable in a notification -- an
// import's error can list hundreds of failed items.
const alertErrorMaxLen = 300

// integrationsLink is where staff check and fix a connection.
const integrationsLink = "/integrations"

const (
	runStatusSuccess = "success"
	runStatusError   = "error"
)

// syncRunRecorder is SyncScheduleRepository narrowed to point of use.
type syncRunRecorder interface {
	RecordRunResult(ctx context.Context, id string, ranAt time.Time, status, errMsg string, nextRunAt time.Time, cursorAt time.Time, advanceLastRunAt bool) error
	RecordHealth(ctx context.Context, id string, failingSince *time.Time, failureAlerted bool) error
}

type connectionFinder interface {
	FindByID(ctx context.Context, id string) (*integrationsDomain.Connection, error)
}

type RecordSyncRunInput struct {
	Schedule channelsDomain.SyncSchedule
	RanAt    time.Time
	// CursorAt is the runner's own LastRunAt cursor; nil = RanAt.
	CursorAt *time.Time
	// RunErr is the run's failure, nil on success.
	RunErr error
}

// RecordSyncRunUseCase stores one scheduled sync run's outcome and tells
// staff when a sync keeps failing (channels.SyncFailureAlertAfter) and
// when it recovers -- 2026-10-31..11-03 a store's 403 failed every run for
// three days and only the scheduler log knew. The alert is recorded as an
// event in the same transaction as the run; the bell/email notification
// is best-effort afterwards, so a notification failure never loses the run.
type RecordSyncRunUseCase struct {
	db        *gorm.DB
	schedules syncRunRecorder
	conns     connectionFinder
	outbox    eventrecord.Enqueuer
	notifier  notifier
}

func NewRecordSyncRunUseCase(db *gorm.DB, schedules syncRunRecorder, conns connectionFinder, outbox eventrecord.Enqueuer, notifier notifier) *RecordSyncRunUseCase {
	return &RecordSyncRunUseCase{db: db, schedules: schedules, conns: conns, outbox: outbox, notifier: notifier}
}

func (uc *RecordSyncRunUseCase) Execute(ctx context.Context, in RecordSyncRunInput) error {
	s := in.Schedule
	status, errMsg := runStatusSuccess, ""
	if in.RunErr != nil {
		status, errMsg = runStatusError, in.RunErr.Error()
	}
	cursorAt := in.RanAt
	if in.CursorAt != nil {
		cursorAt = *in.CursorAt
	}
	nextRunAt := in.RanAt.Add(time.Duration(s.IntervalMinutes) * time.Minute)

	streakStart, wasAlerted := s.FailingSince, s.FailureAlerted
	change := s.ApplyRunOutcome(in.RanAt, in.RunErr != nil)
	healthChanged := wasAlerted != s.FailureAlerted || (streakStart == nil) != (s.FailingSince == nil)

	var store string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.schedules.RecordRunResult(ctx, s.ID, in.RanAt, status, errMsg, nextRunAt, cursorAt, status == runStatusSuccess); err != nil {
			return err
		}
		if healthChanged {
			if err := uc.schedules.RecordHealth(ctx, s.ID, s.FailingSince, s.FailureAlerted); err != nil {
				return err
			}
		}
		if change == channelsDomain.SyncHealthUnchanged {
			return nil
		}
		conn, err := uc.conns.FindByID(ctx, s.ConnectionID)
		if err != nil {
			return err
		}
		store = conn.StoreURL.Host()
		return uc.recordHealthEvent(ctx, s, change, streakStart, errMsg)
	})
	if err != nil {
		return fmt.Errorf("recordSyncRun: %w", err)
	}
	if change != channelsDomain.SyncHealthUnchanged {
		uc.notify(ctx, change, store, errMsg)
	}
	return nil
}

func (uc *RecordSyncRunUseCase) recordHealthEvent(ctx context.Context, s channelsDomain.SyncSchedule, change channelsDomain.SyncHealthChange, streakStart *time.Time, errMsg string) error {
	data := eventrecord.Data{
		"connection_id": s.ConnectionID, "entity_type": string(s.EntityType), "direction": string(s.Direction),
	}
	action := events.SyncScheduleRecovered
	if change == channelsDomain.SyncHealthFailing {
		action = events.SyncScheduleFailureAlerted
		streakStart = s.FailingSince
		data["error"] = errMsg
	}
	if streakStart != nil {
		data["failing_since"] = streakStart.UTC()
	}
	return eventrecord.Record(ctx, uc.outbox, action, s.ID, data)
}

func (uc *RecordSyncRunUseCase) notify(ctx context.Context, change channelsDomain.SyncHealthChange, store, errMsg string) {
	in := appnotifications.CreateNotificationInput{
		Type: notificationsDomain.TypeChannelSyncRecovered, Params: map[string]any{"store": store}, Link: integrationsLink,
	}
	if change == channelsDomain.SyncHealthFailing {
		in.Type = notificationsDomain.TypeChannelSyncFailing
		in.Params["error"] = truncateRunes(errMsg, alertErrorMaxLen)
	}
	if _, err := uc.notifier.Execute(ctx, in); err != nil {
		log.Printf("recordSyncRun: notify %s: %v", in.Type, err)
	}
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}
