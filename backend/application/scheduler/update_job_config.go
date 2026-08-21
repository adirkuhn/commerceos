package scheduler

import (
	"context"
	"fmt"

	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// UpdateJobConfigUseCase is the super_admin "Scheduled Jobs" admin page's
// edit action -- lets enabled/interval_minutes be changed for an existing
// global cron job without a redeploy (previously only settable via a
// one-time migration seed row, see 085_scheduled_jobs.sql).
type UpdateJobConfigUseCase struct {
	db      *gorm.DB
	jobRepo schedulerDomain.JobRepository
}

func NewUpdateJobConfigUseCase(db *gorm.DB, jobRepo schedulerDomain.JobRepository) *UpdateJobConfigUseCase {
	return &UpdateJobConfigUseCase{db: db, jobRepo: jobRepo}
}

func (uc *UpdateJobConfigUseCase) Execute(ctx context.Context, jobKey string, enabled bool, intervalMinutes int) error {
	if intervalMinutes < 1 {
		return schedulerDomain.ErrInvalidIntervalMinutes
	}
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		return uc.jobRepo.UpdateConfig(ctx, jobKey, enabled, intervalMinutes)
	})
	if err != nil {
		return fmt.Errorf("updateJobConfig: %w", err)
	}
	return nil
}
