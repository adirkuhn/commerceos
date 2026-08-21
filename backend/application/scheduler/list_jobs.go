package scheduler

import (
	"context"
	"fmt"

	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListJobsUseCase is the super_admin "Scheduled Jobs" admin page's read
// side -- every global cron job (see domain/scheduler.Job), cross-tenant by
// nature (one run touches every tenant), so no tenant filter to apply.
type ListJobsUseCase struct {
	db      *gorm.DB
	jobRepo schedulerDomain.JobRepository
}

func NewListJobsUseCase(db *gorm.DB, jobRepo schedulerDomain.JobRepository) *ListJobsUseCase {
	return &ListJobsUseCase{db: db, jobRepo: jobRepo}
}

func (uc *ListJobsUseCase) Execute(ctx context.Context) ([]schedulerDomain.Job, error) {
	var jobs []schedulerDomain.Job
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		var err error
		jobs, err = uc.jobRepo.ListAll(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listJobs: %w", err)
	}
	return jobs, nil
}
