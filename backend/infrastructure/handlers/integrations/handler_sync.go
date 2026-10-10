package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListSyncSchedules(c *gin.Context, id genintegrations.Id) {
	schedules, err := h.listSyncSchedules.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncSchedule, len(schedules))
	for i, s := range schedules {
		resp[i] = toGenSyncSchedule(s)
	}
	middleware.OK(c, resp)
}

// UpsertSyncSchedules saves the whole sync-schedule grid in one
// all-or-nothing batch (see UpsertSyncSchedulesUseCase's doc comment) --
// either every row in the request saves, or none of them do, so a failing
// row can never silently block a different row's change from being
// attempted at all.
func (h *Handler) UpsertSyncSchedules(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpsertSyncSchedulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	inputs := make([]appchannels.UpsertSyncScheduleInput, len(req.Schedules))
	for i, s := range req.Schedules {
		inputs[i] = appchannels.UpsertSyncScheduleInput{
			ConnectionID:       id,
			EntityType:         channelsDomain.SyncEntityType(s.EntityType),
			Direction:          channelsDomain.SyncDirection(s.Direction),
			Enabled:            s.Enabled,
			IntervalMinutes:    s.IntervalMinutes,
			DefaultWarehouseID: s.DefaultWarehouseId,
			DefaultCurrency:    s.DefaultCurrency,
		}
	}
	updated, err := h.upsertSyncSchedules.Execute(c.Request.Context(), appchannels.UpsertSyncSchedulesInput{Schedules: inputs})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncSchedule, len(updated))
	for i, s := range updated {
		resp[i] = toGenSyncSchedule(s)
	}
	middleware.OK(c, resp)
}

func (h *Handler) GetProductChannelListing(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	listing, err := h.getProductChannelListing.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListing(*listing))
}

func (h *Handler) ListProductChannelListings(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	listings, err := h.listProductChannelListings.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListings(listings))
}

func (h *Handler) ListSyncLogs(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	entries, err := h.listSyncLogs.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncLogEntry, len(entries))
	for i, e := range entries {
		resp[i] = toGenSyncLogEntry(e)
	}
	middleware.OK(c, resp)
}

func (h *Handler) ListAllSyncLogs(c *gin.Context, params genintegrations.ListAllSyncLogsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	entries, total, err := h.listAllSyncLogs.Execute(c.Request.Context(), page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncLogEntry, len(entries))
	for i, e := range entries {
		gen := toGenSyncLogEntry(e.SyncLogEntry)
		if e.ProductName != "" {
			gen.ProductName = &e.ProductName
		}
		if e.StoreURL != "" {
			gen.StoreUrl = &e.StoreURL
		}
		resp[i] = gen
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, genintegrations.SyncLogEntryPageResponse{Data: &resp, Total: &totalInt})
}

func (h *Handler) ListSyncLogsNeedingAttention(c *gin.Context, params genintegrations.ListSyncLogsNeedingAttentionParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	entries, total, err := h.listSyncLogsNeedingAttention.Execute(c.Request.Context(), page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncLogEntry, len(entries))
	for i, e := range entries {
		gen := toGenSyncLogEntry(e.SyncLogEntry)
		if e.ProductName != "" {
			gen.ProductName = &e.ProductName
		}
		if e.StoreURL != "" {
			gen.StoreUrl = &e.StoreURL
		}
		resp[i] = gen
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, genintegrations.SyncLogEntryPageResponse{Data: &resp, Total: &totalInt})
}

func (h *Handler) RetryFailedSync(c *gin.Context, id genintegrations.Id) {
	if err := h.retryFailedSync.Execute(c.Request.Context(), id); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genintegrations.EmptyResponse{})
}

func (h *Handler) DismissFailedSync(c *gin.Context, id genintegrations.Id) {
	if err := h.dismissFailedSync.Execute(c.Request.Context(), id); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genintegrations.EmptyResponse{})
}

func (h *Handler) ResyncProductToChannels(c *gin.Context, productId genintegrations.ProductId) {
	if err := h.resyncProductToChannels.Execute(c.Request.Context(), productId); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genintegrations.EmptyResponse{})
}

func toGenListing(l channelsDomain.ProductChannelListing) genintegrations.ProductChannelListing {
	isVariation := l.IsVariation
	note := l.NotForSaleNote
	return genintegrations.ProductChannelListing{
		Id:             l.ID,
		ProductId:      l.ProductID,
		ConnectionId:   l.ConnectionID,
		Provider:       l.Provider,
		ExternalId:     l.ExternalID,
		VariantId:      l.VariantID,
		IsVariation:    &isVariation,
		SyncStatus:     genintegrations.ProductChannelListingSyncStatus(l.SyncStatus),
		LastSyncedAt:   l.LastSyncedAt,
		NotForSaleNote: &note,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}

func toGenListings(ls []channelsDomain.ProductChannelListing) []genintegrations.ProductChannelListing {
	out := make([]genintegrations.ProductChannelListing, len(ls))
	for i, l := range ls {
		out[i] = toGenListing(l)
	}
	return out
}

func toGenSyncLogEntry(e channelsDomain.SyncLogEntry) genintegrations.SyncLogEntry {
	changes := make(map[string]genintegrations.FieldChange, len(e.Changes))
	for k, v := range e.Changes {
		changes[k] = genintegrations.FieldChange{From: v.From, To: v.To}
	}
	fields := e.Fields
	if fields == nil {
		fields = map[string]string{}
	}
	return genintegrations.SyncLogEntry{
		Id: e.ID, ListingId: e.ListingID, ProductId: e.ProductID, ConnectionId: e.ConnectionID,
		Provider: e.Provider, ExternalId: e.ExternalID,
		Direction: genintegrations.SyncLogEntryDirection(e.Direction), Status: genintegrations.SyncLogEntryStatus(e.Status),
		ErrorMessage: e.ErrorMessage, Fields: fields, Changes: changes, SyncedAt: e.SyncedAt, DismissedAt: e.DismissedAt,
	}
}

func toGenSyncSchedule(s channelsDomain.SyncSchedule) genintegrations.SyncSchedule {
	return genintegrations.SyncSchedule{
		EntityType:         genintegrations.SyncScheduleEntityType(s.EntityType),
		Direction:          genintegrations.SyncScheduleDirection(s.Direction),
		Enabled:            s.Enabled,
		IntervalMinutes:    s.IntervalMinutes,
		DefaultWarehouseId: s.DefaultWarehouseID,
		DefaultCurrency:    s.DefaultCurrency,
		LastRunAt:          s.LastRunAt,
		LastRunStatus:      genintegrations.SyncScheduleLastRunStatus(s.LastRunStatus),
		LastRunError:       s.LastRunError,
	}
}
