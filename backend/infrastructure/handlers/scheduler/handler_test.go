package scheduler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genscheduler "github.com/adirkuhn/commerceos/api/scheduler"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	schedulerhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/scheduler"
)

type stubListJobs struct {
	out []schedulerDomain.Job
	err error
}

func (s *stubListJobs) Execute(_ context.Context) ([]schedulerDomain.Job, error) { return s.out, s.err }

type stubRunJobNow struct {
	err       error
	gotJobKey string
}

func (s *stubRunJobNow) Execute(_ context.Context, jobKey string) error {
	s.gotJobKey = jobKey
	return s.err
}

type stubUpdateJobConfig struct {
	err             error
	gotJobKey       string
	gotEnabled      bool
	gotIntervalMins int
}

func (s *stubUpdateJobConfig) Execute(_ context.Context, jobKey string, enabled bool, intervalMinutes int) error {
	s.gotJobKey, s.gotEnabled, s.gotIntervalMins = jobKey, enabled, intervalMinutes
	return s.err
}

type stubListSyncSchedules struct {
	out []appchannels.SyncScheduleWithConnection
	err error
}

func (s *stubListSyncSchedules) Execute(_ context.Context) ([]appchannels.SyncScheduleWithConnection, error) {
	return s.out, s.err
}

type stubRunSyncScheduleNow struct {
	err   error
	gotID string
}

func (s *stubRunSyncScheduleNow) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

func newTestRouter(list *stubListJobs, run *stubRunJobNow, listSync *stubListSyncSchedules, runSync *stubRunSyncScheduleNow) *gin.Engine {
	h := schedulerhandler.NewHandler(list, run, &stubUpdateJobConfig{}, listSync, runSync)
	r := gin.New()
	genscheduler.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

// newUpdateJobConfigTestRouter builds a router for the PATCH
// /scheduler/jobs/{jobKey} route specifically — every other use case is a
// harmless zero-value stub.
func newUpdateJobConfigTestRouter(update *stubUpdateJobConfig) *gin.Engine {
	h := schedulerhandler.NewHandler(&stubListJobs{}, &stubRunJobNow{}, update, &stubListSyncSchedules{}, &stubRunSyncScheduleNow{})
	r := gin.New()
	genscheduler.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListJobs_success(t *testing.T) {
	list := &stubListJobs{out: []schedulerDomain.Job{
		{JobKey: "cost_price_recalculation", Enabled: true, IntervalMinutes: 1440, NextRunAt: time.Now()},
	}}
	w := doRequest(newTestRouter(list, &stubRunJobNow{}, &stubListSyncSchedules{}, &stubRunSyncScheduleNow{}), "GET", "/api/v1/scheduler/jobs", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cost_price_recalculation")
}

func TestRunJobNow_success(t *testing.T) {
	run := &stubRunJobNow{}
	w := doRequest(newTestRouter(&stubListJobs{}, run, &stubListSyncSchedules{}, &stubRunSyncScheduleNow{}), "POST", "/api/v1/scheduler/jobs/expired_batches/run-now", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "expired_batches", run.gotJobKey)
}

func TestRunJobNow_notFound(t *testing.T) {
	run := &stubRunJobNow{err: schedulerDomain.ErrJobNotFound}
	w := doRequest(newTestRouter(&stubListJobs{}, run, &stubListSyncSchedules{}, &stubRunSyncScheduleNow{}), "POST", "/api/v1/scheduler/jobs/unknown/run-now", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateJobConfig_success(t *testing.T) {
	update := &stubUpdateJobConfig{}
	w := doRequest(newUpdateJobConfigTestRouter(update), "PATCH", "/api/v1/scheduler/jobs/resync_all_products_to_channels",
		genscheduler.UpdateJobConfigRequest{Enabled: true, IntervalMinutes: 60})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "resync_all_products_to_channels", update.gotJobKey)
	assert.True(t, update.gotEnabled)
	assert.Equal(t, 60, update.gotIntervalMins)
}

func TestUpdateJobConfig_notFound(t *testing.T) {
	update := &stubUpdateJobConfig{err: schedulerDomain.ErrJobNotFound}
	w := doRequest(newUpdateJobConfigTestRouter(update), "PATCH", "/api/v1/scheduler/jobs/unknown",
		genscheduler.UpdateJobConfigRequest{Enabled: true, IntervalMinutes: 60})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateJobConfig_invalidInterval(t *testing.T) {
	update := &stubUpdateJobConfig{err: schedulerDomain.ErrInvalidIntervalMinutes}
	w := doRequest(newUpdateJobConfigTestRouter(update), "PATCH", "/api/v1/scheduler/jobs/expired_batches",
		genscheduler.UpdateJobConfigRequest{Enabled: true, IntervalMinutes: 0})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListSyncSchedules_success(t *testing.T) {
	listSync := &stubListSyncSchedules{out: []appchannels.SyncScheduleWithConnection{
		{Schedule: channelsDomain.SyncSchedule{ID: "sched-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport}, ConnectionStoreURL: "https://mystore.com"},
	}}
	w := doRequest(newTestRouter(&stubListJobs{}, &stubRunJobNow{}, listSync, &stubRunSyncScheduleNow{}), "GET", "/api/v1/scheduler/sync-schedules", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mystore.com")
}

func TestRunSyncScheduleNow_success(t *testing.T) {
	runSync := &stubRunSyncScheduleNow{}
	w := doRequest(newTestRouter(&stubListJobs{}, &stubRunJobNow{}, &stubListSyncSchedules{}, runSync), "POST", "/api/v1/scheduler/sync-schedules/sched-1/run-now", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sched-1", runSync.gotID)
}

func TestRunSyncScheduleNow_notFound(t *testing.T) {
	runSync := &stubRunSyncScheduleNow{err: channelsDomain.ErrSyncScheduleNotFound}
	w := doRequest(newTestRouter(&stubListJobs{}, &stubRunJobNow{}, &stubListSyncSchedules{}, runSync), "POST", "/api/v1/scheduler/sync-schedules/missing/run-now", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
