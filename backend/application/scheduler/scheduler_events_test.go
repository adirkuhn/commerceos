package scheduler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

func jobRepoWith(job schedulerDomain.Job) *stubJobRepo {
	return &stubJobRepo{byKey: map[string]schedulerDomain.Job{job.JobKey: job}}
}

func TestUpdateJobConfig_RecordsChangedConfig(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := jobRepoWith(schedulerDomain.Job{ID: 7, JobKey: "expired_batches", Enabled: true, IntervalMinutes: 30})

	require.NoError(t, appscheduler.NewUpdateJobConfigUseCase(newMockGormDB(t), repo, outbox).Execute(context.Background(), "expired_batches", false, 30))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "scheduler.job.updated", e.Subject)
	assert.Equal(t, "expired_batches", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"enabled": {true, false}}, e.Changes())
	assert.Equal(t, map[string]any{"job_key": "expired_batches"}, e.Data())
}

func TestUpdateJobConfig_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	repo := jobRepoWith(schedulerDomain.Job{JobKey: "expired_batches", IntervalMinutes: 30})

	err := appscheduler.NewUpdateJobConfigUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{Err: boom}).Execute(context.Background(), "expired_batches", true, 60)

	assert.ErrorIs(t, err, boom)
}
