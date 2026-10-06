package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpsertSyncSchedulesInput struct {
	Schedules []UpsertSyncScheduleInput
}

type UpsertSyncSchedulesUseCase struct {
	db     *gorm.DB
	repo   channelsDomain.SyncScheduleRepository
	outbox eventrecord.Enqueuer
}

func NewUpsertSyncSchedulesUseCase(db *gorm.DB, repo channelsDomain.SyncScheduleRepository, outbox eventrecord.Enqueuer) *UpsertSyncSchedulesUseCase {
	return &UpsertSyncSchedulesUseCase{db: db, repo: repo, outbox: outbox}
}

// Execute is the settings page's "Save" action for the whole sync-schedule
// grid (products/stock/orders x import/export) in one PUT -- validates
// every row with zero writes before persisting anything, then persists the
// whole batch in a single transaction. Both halves matter: without the
// validate-first pass, one row's error could stop a later row (the one the
// user actually meant to change) from ever being attempted, while earlier
// rows in the same request had already landed -- confusing "I changed
// Orders and hit Save, but it didn't stick" behavior with no clear cause.
// Now it's all-or-nothing: either every row in the request saves, or none
// of them do, with an error naming exactly which row is invalid.
func (uc *UpsertSyncSchedulesUseCase) Execute(ctx context.Context, in UpsertSyncSchedulesInput) ([]channelsDomain.SyncSchedule, error) {
	for _, s := range in.Schedules {
		if err := validateUpsertSyncScheduleInput(s); err != nil {
			return nil, fmt.Errorf("upsertSyncSchedules: %s/%s: %w", s.EntityType, s.Direction, err)
		}
	}

	schedules := make([]channelsDomain.SyncSchedule, len(in.Schedules))
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		for i, in := range in.Schedules {
			schedules[i] = buildSyncSchedule(ctx, in, now)
			if err := uc.repo.Upsert(ctx, &schedules[i]); err != nil {
				return err
			}
			if err := eventrecord.Record(ctx, uc.outbox, events.SyncScheduleSaved, schedules[i].ID, syncScheduleData(schedules[i])); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("upsertSyncSchedules: %w", err)
	}
	return schedules, nil
}
