package scheduler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
)

var errBoom = errors.New("boom")

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// tenant.WithTenant's Begin/set_config/Commit sequence -- same pattern as
// application/inventory's own testutil_test.go.
func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newMockGormDBN(t, 1)
}

func newMockGormDBN(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}

	return gormDB
}

type stubJobRepo struct {
	all     []schedulerDomain.Job
	listErr error

	due    []schedulerDomain.Job
	dueErr error

	byKey   map[string]schedulerDomain.Job
	findErr error

	recordCalls []recordRunCall
	recordErr   error

	triggeredKeys []string
	triggerErr    error
}

type recordRunCall struct {
	JobKey    string
	RanAt     time.Time
	Status    string
	ErrMsg    string
	NextRunAt time.Time
}

func (s *stubJobRepo) ListAll(_ context.Context) ([]schedulerDomain.Job, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.all, nil
}

func (s *stubJobRepo) ListDue(_ context.Context, _ time.Time) ([]schedulerDomain.Job, error) {
	if s.dueErr != nil {
		return nil, s.dueErr
	}
	return s.due, nil
}

func (s *stubJobRepo) FindByKey(_ context.Context, jobKey string) (*schedulerDomain.Job, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if job, ok := s.byKey[jobKey]; ok {
		return &job, nil
	}
	return nil, schedulerDomain.ErrJobNotFound
}

func (s *stubJobRepo) RecordRunResult(_ context.Context, jobKey string, ranAt time.Time, status, errMsg string, nextRunAt time.Time) error {
	s.recordCalls = append(s.recordCalls, recordRunCall{JobKey: jobKey, RanAt: ranAt, Status: status, ErrMsg: errMsg, NextRunAt: nextRunAt})
	return s.recordErr
}

func (s *stubJobRepo) TriggerNow(_ context.Context, jobKey string) error {
	if s.triggerErr != nil {
		return s.triggerErr
	}
	s.triggeredKeys = append(s.triggeredKeys, jobKey)
	return nil
}

type stubJobRunner struct {
	err   error
	calls int
}

func (s *stubJobRunner) Run(_ context.Context) error {
	s.calls++
	return s.err
}
