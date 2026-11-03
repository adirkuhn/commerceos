package scheduler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

// Every known job can be queued from the admin -- the API no longer needs
// a runner of its own (the 2026-11-03 bug: its partial copy lacked
// promotion_activation, reconcile_fulfilled_orders, ...).
func TestTriggerJobNow_queuesEveryKnownJob(t *testing.T) {
	for _, key := range schedulerDomain.JobKeys() {
		t.Run(key, func(t *testing.T) {
			jobRepo := &stubJobRepo{byKey: map[string]schedulerDomain.Job{key: {JobKey: key}}}
			outbox := &eventrecordtest.Outbox{}
			uc := appscheduler.NewTriggerJobNowUseCase(newMockGormDBN(t, 1), jobRepo, outbox)

			require.NoError(t, uc.Execute(context.Background(), key))

			require.Equal(t, []string{key}, jobRepo.triggeredKeys)
			e, err := outbox.Only()
			require.NoError(t, err)
			require.Equal(t, "scheduler.job.run_requested", e.Subject)
			require.Equal(t, key, e.Payload["entity_id"])
		})
	}
}

func TestTriggerJobNow_unknownJobIsNotFound(t *testing.T) {
	jobRepo := &stubJobRepo{triggerErr: schedulerDomain.ErrJobNotFound}
	outbox := &eventrecordtest.Outbox{}
	uc := appscheduler.NewTriggerJobNowUseCase(newMockGormDBN(t, 1), jobRepo, outbox)

	err := uc.Execute(context.Background(), "not-a-real-job")

	require.ErrorIs(t, err, schedulerDomain.ErrJobNotFound)
	_, onlyErr := outbox.Only()
	require.Error(t, onlyErr, "nothing queued -> no event")
}
