package scheduler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genscheduler "github.com/adirkuhn/commerceos/api/scheduler"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	bolinfra "github.com/adirkuhn/commerceos/infrastructure/bol"
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
func Register(rg gin.IRouter, db *gorm.DB, wooFactory integrationsDomain.WooClientFactory, encryptionSecret string) {
	jobRepo := repositories.NewScheduledJobRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	connRepo := repositories.NewConnectionRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	productRepo := repositories.NewProductRepo(db)
	categoryRepo := repositories.NewCategoryRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	costPriceHistoryRepo := repositories.NewSKUCostPriceHistoryRepo(db)
	priceHistoryRepo := repositories.NewSKUPriceHistoryRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)
	syncLogRepo := repositories.NewSyncLogRepo(db)
	mediaRepo := repositories.NewMediaRepo(db)
	salesChannelRepo := repositories.NewSalesChannelRepo(db)
	salesChannelOverrideRepo := repositories.NewSalesChannelProductOverrideRepo(db)
	notificationRepo := repositories.NewNotificationRepo(db)

	expireSalePriceUC := apppms.NewExpireSKUSalePriceUseCase(db, variantRepo, skuRepo, outboxRepo, priceHistoryRepo)
	expireBatchUC := appinventory.NewExpireBatchUseCase(db, batchRepo, outboxRepo)
	recalculateCostPricesUC := apppms.NewRecalculateCostPricesUseCase(
		db, skuRepo, variantRepo, productRepo, categoryRepo, settingsRepo, costPriceHistoryRepo,
		appinventory.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
		appprocurement.NewListReceivedCostLinesForSKUUseCase(db, lineItemRepo),
	)
	stockLevelUC := appinventory.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo, settingsRepo)
	categoryLinkRepo := repositories.NewExternalCategoryLinkRepo(db)
	resolveCategoryLinkUC := appchannels.NewResolveCategoryLinkUseCase(db, categoryLinkRepo, categoryRepo)
	pushProductToWooUC := appchannels.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, encryptionSecret, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, stockLevelUC, mediaRepo, resolveCategoryLinkUC,
	)
	bolOverrideRepo := repositories.NewBolProductOverrideRepo(db)
	bolContentRepo := repositories.NewBolProductContentRepo(db)
	pushProductToBolUC := appchannels.NewPushProductToBolUseCase(
		db, connRepo, bolinfra.NewClientFactory(), encryptionSecret, productRepo, variantRepo, skuRepo, listingRepo, bolOverrideRepo, bolContentRepo, syncLogRepo, stockLevelUC,
		appnotifications.NewCreateNotificationUseCase(db, notificationRepo),
	)
	syncStockToChannelsUC := appchannels.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, syncLogRepo, connRepo, wooFactory, stockLevelUC, encryptionSecret, pushProductToBolUC,
	)
	syncProductToChannelsUC := appchannels.NewSyncProductToChannelsUseCase(
		db, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, connRepo, wooFactory, encryptionSecret, pushProductToWooUC, mediaRepo, resolveCategoryLinkUC, pushProductToBolUC, pushProductToBolUC, syncStockToChannelsUC,
	)
	enrollProductInChannelsUC := appchannels.NewEnrollProductInChannelsUseCase(
		db, productRepo, salesChannelRepo, salesChannelOverrideRepo, pushProductToWooUC, pushProductToBolUC,
	)
	resyncProductToChannelsUC := appchannels.NewResyncProductToChannelsUseCase(db, variantRepo, skuRepo, syncProductToChannelsUC, syncStockToChannelsUC, enrollProductInChannelsUC)

	runners := map[string]appscheduler.JobRunner{
		schedulerDomain.JobExpiredSalePrices:           appscheduler.NewRunExpiredSalePricesUseCase(db, skuRepo, expireSalePriceUC),
		schedulerDomain.JobExpiredBatches:              appscheduler.NewRunExpiredBatchesUseCase(db, batchRepo, expireBatchUC),
		schedulerDomain.JobCostPriceRecalculation:      appscheduler.NewRunCostPriceRecalculationUseCase(db, skuRepo, recalculateCostPricesUC),
		schedulerDomain.JobResyncAllProductsToChannels: appscheduler.NewRunResyncProductsToChannelsUseCase(db, listingRepo, resyncProductToChannelsUC),
	}

	h := NewHandler(
		appscheduler.NewListJobsUseCase(db, jobRepo),
		appscheduler.NewRunJobNowUseCase(db, jobRepo, runners),
		appscheduler.NewUpdateJobConfigUseCase(db, jobRepo),
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
	c.JSON(http.StatusOK, genscheduler.EmptyResponse{})
}

func (h *Handler) UpdateJobConfig(c *gin.Context, jobKey genscheduler.JobKey) {
	var req genscheduler.UpdateJobConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
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
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, schedulerDomain.ErrInvalidIntervalMinutes):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	}
	return err
}
