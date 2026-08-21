package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// TriggerSyncScheduleNowUseCase is the super_admin "Scheduled Jobs" admin
// page's run-now action for a per-tenant Woo sync schedule -- pulls
// NextRunAt to now so cmd/scheduler's own poll tick (within pollInterval)
// picks it up. Deliberately not synchronous: a Woo import can page through
// thousands of orders, that belongs on the scheduler process's polling
// loop, not blocking an HTTP request.
type TriggerSyncScheduleNowUseCase struct {
	db           *gorm.DB
	scheduleRepo channelsDomain.SyncScheduleRepository
}

func NewTriggerSyncScheduleNowUseCase(db *gorm.DB, scheduleRepo channelsDomain.SyncScheduleRepository) *TriggerSyncScheduleNowUseCase {
	return &TriggerSyncScheduleNowUseCase{db: db, scheduleRepo: scheduleRepo}
}

func (uc *TriggerSyncScheduleNowUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		return uc.scheduleRepo.TriggerNow(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("triggerSyncScheduleNow: %w", err)
	}
	return nil
}
