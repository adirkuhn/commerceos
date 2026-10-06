package scheduler_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

func TestUpdateJobConfig_updatesEnabledAndInterval(t *testing.T) {
	jobRepo := &stubJobRepo{byKey: map[string]schedulerDomain.Job{"resync_all_products_to_channels": {JobKey: "resync_all_products_to_channels"}}}
	uc := appscheduler.NewUpdateJobConfigUseCase(newMockGormDB(t), jobRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "resync_all_products_to_channels", true, 60)

	require.NoError(t, err)
	require.Equal(t, []updateConfigCall{{JobKey: "resync_all_products_to_channels", Enabled: true, IntervalMinutes: 60}}, jobRepo.updateConfigCalls)
}

func TestUpdateJobConfig_rejectsIntervalBelowOne(t *testing.T) {
	jobRepo := &stubJobRepo{}
	uc := appscheduler.NewUpdateJobConfigUseCase(newMockGormDBN(t, 0), jobRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "expired_batches", true, 0)

	require.ErrorIs(t, err, schedulerDomain.ErrInvalidIntervalMinutes)
	require.Empty(t, jobRepo.updateConfigCalls, "must not touch the repo for an invalid interval")
}

func TestUpdateJobConfig_propagatesJobNotFound(t *testing.T) {
	jobRepo := &stubJobRepo{updateConfigErr: schedulerDomain.ErrJobNotFound}
	uc := appscheduler.NewUpdateJobConfigUseCase(newMockGormDB(t), jobRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing_job", true, 60)

	require.ErrorIs(t, err, schedulerDomain.ErrJobNotFound)
}
