package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

func TestListJobs_returnsEveryJobCrossTenant(t *testing.T) {
	jobRepo := &stubJobRepo{all: []schedulerDomain.Job{
		{JobKey: "expired_sale_prices", Enabled: true, NextRunAt: time.Now()},
		{JobKey: "cost_price_recalculation", Enabled: true, NextRunAt: time.Now()},
	}}
	uc := appscheduler.NewListJobsUseCase(newMockGormDB(t), jobRepo)

	jobs, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, jobs, 2)
}

func TestListJobs_propagatesARepoError(t *testing.T) {
	jobRepo := &stubJobRepo{listErr: errBoom}
	uc := appscheduler.NewListJobsUseCase(newMockGormDB(t), jobRepo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
