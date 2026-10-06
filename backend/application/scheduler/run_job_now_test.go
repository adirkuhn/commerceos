package scheduler_test

import (
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

func TestRunJobNow_runsAndRecordsSuccess(t *testing.T) {
	jobRepo := &stubJobRepo{byKey: map[string]schedulerDomain.Job{
		"expired_sale_prices": {JobKey: "expired_sale_prices", IntervalMinutes: 1},
	}}
	runner := &stubJobRunner{}
	uc := appscheduler.NewRunJobNowUseCase(newMockGormDBN(t, 2), jobRepo, map[string]appscheduler.JobRunner{"expired_sale_prices": runner}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "expired_sale_prices")

	require.NoError(t, err)
	require.Equal(t, 1, runner.calls)
	require.Len(t, jobRepo.recordCalls, 1)
	require.Equal(t, "success", jobRepo.recordCalls[0].Status)
	require.Empty(t, jobRepo.recordCalls[0].ErrMsg)
	require.WithinDuration(t, jobRepo.recordCalls[0].RanAt.Add(time.Minute), jobRepo.recordCalls[0].NextRunAt, time.Second)
}

func TestRunJobNow_recordsFailureButStillReturnsTheRunError(t *testing.T) {
	jobRepo := &stubJobRepo{byKey: map[string]schedulerDomain.Job{
		"expired_batches": {JobKey: "expired_batches", IntervalMinutes: 1},
	}}
	runner := &stubJobRunner{err: errors.New("db unreachable")}
	uc := appscheduler.NewRunJobNowUseCase(newMockGormDBN(t, 2), jobRepo, map[string]appscheduler.JobRunner{"expired_batches": runner}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "expired_batches")

	require.Error(t, err)
	require.Len(t, jobRepo.recordCalls, 1)
	require.Equal(t, "error", jobRepo.recordCalls[0].Status)
	require.Equal(t, "db unreachable", jobRepo.recordCalls[0].ErrMsg)
}

func TestRunJobNow_unknownJobKeyIsNotFound(t *testing.T) {
	jobRepo := &stubJobRepo{}
	uc := appscheduler.NewRunJobNowUseCase(newMockGormDB(t), jobRepo, map[string]appscheduler.JobRunner{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "not-a-real-job")

	require.ErrorIs(t, err, schedulerDomain.ErrJobNotFound)
}

func TestRunJobNow_RecordsTheTriggeredRun(t *testing.T) {
	jobRepo := &stubJobRepo{byKey: map[string]schedulerDomain.Job{"expired_batches": {JobKey: "expired_batches", IntervalMinutes: 1}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appscheduler.NewRunJobNowUseCase(newMockGormDBN(t, 2), jobRepo,
		map[string]appscheduler.JobRunner{"expired_batches": &stubJobRunner{err: errors.New("db locked")}}, outbox)

	_ = uc.Execute(context.Background(), "expired_batches")

	e, err := outbox.Only()
	require.NoError(t, err)
	require.Equal(t, "scheduler.job.run_triggered", e.Subject)
	require.Equal(t, "expired_batches", e.Payload["entity_id"])
	require.Equal(t, map[string]any{"status": "error", "error": "db locked"}, e.Data())
}
