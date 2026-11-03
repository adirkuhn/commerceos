package scheduler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genscheduler "github.com/adirkuhn/commerceos/api/scheduler"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listJobsUC interface {
	Execute(ctx context.Context) ([]schedulerDomain.Job, error)
}
type runJobNowUC interface {
	Execute(ctx context.Context, jobKey string) error
}
type updateJobConfigUC interface {
	Execute(ctx context.Context, jobKey string, enabled bool, intervalMinutes int) error
}
type listSyncSchedulesUC interface {
	Execute(ctx context.Context) ([]appchannels.SyncScheduleWithConnection, error)
}
type runSyncScheduleNowUC interface {
	Execute(ctx context.Context, id string) error
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listJobs           listJobsUC
	runJobNow          runJobNowUC
	updateJobConfig    updateJobConfigUC
	listSyncSchedules  listSyncSchedulesUC
	runSyncScheduleNow runSyncScheduleNowUC
}

func NewHandler(listJobs listJobsUC, runJobNow runJobNowUC, updateJobConfig updateJobConfigUC, listSyncSchedules listSyncSchedulesUC, runSyncScheduleNow runSyncScheduleNowUC) *Handler {
	return &Handler{listJobs: listJobs, runJobNow: runJobNow, updateJobConfig: updateJobConfig, listSyncSchedules: listSyncSchedules, runSyncScheduleNow: runSyncScheduleNow}
}

var _ genscheduler.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers must mount rg with
// middleware.RequireRole(middleware.RoleSuperAdmin) already applied --
// every route here is a cross-tenant platform-operational view/action, not
// tenant self-service settings (see api/scheduler/openapi.yaml's doc).
func Register(rg gin.IRouter, db *gorm.DB) {
	jobRepo := repositories.NewScheduledJobRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	connRepo := repositories.NewConnectionRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)

	// Run-now only queues the job: cmd/scheduler owns the one runner
	// registry (see appscheduler.TriggerJobNowUseCase).
	h := NewHandler(
		appscheduler.NewListJobsUseCase(db, jobRepo),
		appscheduler.NewTriggerJobNowUseCase(db, jobRepo, outboxRepo),
		appscheduler.NewUpdateJobConfigUseCase(db, jobRepo, outboxRepo),
		appchannels.NewListAllSyncSchedulesUseCase(db, scheduleRepo, connRepo),
		appchannels.NewTriggerSyncScheduleNowUseCase(db, scheduleRepo),
	)
	genscheduler.RegisterHandlers(rg, h)
}

func (h *Handler) ListJobs(c *gin.Context) {
	jobs, err := h.listJobs.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genscheduler.Job, len(jobs))
	for i, j := range jobs {
		out[i] = genscheduler.Job{
			JobKey: j.JobKey, Enabled: j.Enabled, IntervalMinutes: j.IntervalMinutes,
			LastRunAt: j.LastRunAt, LastRunStatus: genscheduler.JobLastRunStatus(j.LastRunStatus), LastRunError: j.LastRunError,
			NextRunAt: j.NextRunAt,
		}
	}
	c.JSON(http.StatusOK, genscheduler.JobListResponse{Data: &out})
}

func (h *Handler) RunJobNow(c *gin.Context, jobKey genscheduler.JobKey) {
	if err := h.runJobNow.Execute(c.Request.Context(), jobKey); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, genscheduler.EmptyResponse{})
}

func (h *Handler) UpdateJobConfig(c *gin.Context, jobKey genscheduler.JobKey) {
	var req genscheduler.UpdateJobConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if err := h.updateJobConfig.Execute(c.Request.Context(), jobKey, req.Enabled, req.IntervalMinutes); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genscheduler.EmptyResponse{})
}

func (h *Handler) ListSyncSchedules(c *gin.Context) {
	rows, err := h.listSyncSchedules.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genscheduler.SyncScheduleRow, len(rows))
	for i, r := range rows {
		out[i] = genscheduler.SyncScheduleRow{
			Id: r.Schedule.ID, TenantId: r.Schedule.TenantID, ConnectionId: r.Schedule.ConnectionID,
			ConnectionStoreUrl: r.ConnectionStoreURL,
			EntityType:         genscheduler.SyncScheduleRowEntityType(r.Schedule.EntityType),
			Direction:          genscheduler.SyncScheduleRowDirection(r.Schedule.Direction),
			Enabled:            r.Schedule.Enabled, IntervalMinutes: r.Schedule.IntervalMinutes,
			LastRunAt: r.Schedule.LastRunAt, LastRunStatus: genscheduler.SyncScheduleRowLastRunStatus(r.Schedule.LastRunStatus), LastRunError: r.Schedule.LastRunError,
			NextRunAt: r.Schedule.NextRunAt,
		}
	}
	c.JSON(http.StatusOK, genscheduler.SyncScheduleListResponse{Data: &out})
}

func (h *Handler) RunSyncScheduleNow(c *gin.Context, id genscheduler.Id) {
	if err := h.runSyncScheduleNow.Execute(c.Request.Context(), id); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genscheduler.EmptyResponse{})
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, schedulerDomain.ErrJobNotFound), errors.Is(err, channelsDomain.ErrSyncScheduleNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, schedulerDomain.ErrInvalidIntervalMinutes):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	}
	return err
}
