package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListSyncSchedulesUseCase struct {
	db   *gorm.DB
	repo channelsDomain.SyncScheduleRepository
}

func NewListSyncSchedulesUseCase(db *gorm.DB, repo channelsDomain.SyncScheduleRepository) *ListSyncSchedulesUseCase {
	return &ListSyncSchedulesUseCase{db: db, repo: repo}
}

func (uc *ListSyncSchedulesUseCase) Execute(ctx context.Context, connectionID string) ([]channelsDomain.SyncSchedule, error) {
	var schedules []channelsDomain.SyncSchedule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		schedules, err = uc.repo.ListByConnection(ctx, connectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSyncSchedules: %w", err)
	}
	return schedules, nil
}
