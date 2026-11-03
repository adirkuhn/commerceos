// Packing queue + session-lock plumbing, in its own file for the same reason
// as packing_handler.go: keeps handler.go from growing further.
package oms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type markOrderPickedUC interface {
	Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) error
}

type listPackingQueueUC interface {
	Execute(ctx context.Context, page, perPage int) (*appoms.ListQueueResult, error)
}

// lockActorFrom maps the caller's JWT to who is asking for a session lock:
// super_admin/customer_owner/customer_admin may take over a lock someone
// else is actively holding (see domain/oms.SessionLock.CanTakeOver).
func lockActorFrom(claims *middleware.Claims) omsDomain.LockActor {
	if claims == nil {
		return omsDomain.LockActor{}
	}
	switch claims.Role {
	case middleware.RoleSuperAdmin, middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin:
		return omsDomain.LockActor{UserID: claims.UserID, IsAdmin: true}
	default:
		return omsDomain.LockActor{UserID: claims.UserID}
	}
}

func (h *Handler) ListPackingQueue(c *gin.Context, params genoms.ListPackingQueueParams) {
	h.respondWithQueue(c, h.listPackingQueue, params.Page, params.PerPage)
}

func (h *Handler) ListPickingQueue(c *gin.Context, params genoms.ListPickingQueueParams) {
	h.respondWithQueue(c, h.listPickingQueue, params.Page, params.PerPage)
}

func (h *Handler) respondWithQueue(c *gin.Context, uc listPackingQueueUC, pageParam, perPageParam *int) {
	page, perPage := 1, 20
	if pageParam != nil {
		page = *pageParam
	}
	if perPageParam != nil {
		perPage = *perPageParam
	}
	result, err := uc.Execute(c.Request.Context(), page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	queued := make([]omsDomain.SalesOrder, len(result.Entries))
	for i, e := range result.Entries {
		queued[i] = e.Order
	}
	orders, ok := h.genSalesOrders(c, queued...)
	if !ok {
		return
	}
	entries := make([]genoms.QueueEntry, len(result.Entries))
	for i, e := range result.Entries {
		entries[i] = genoms.QueueEntry{SalesOrder: orders[i], Lock: toGenQueueLock(e.Lock), InPacking: e.InPacking}
	}
	resp := genoms.OrderQueue{Entries: entries, Total: result.Total}
	c.JSON(http.StatusOK, genoms.OrderQueueResponse{Data: &resp})
}

func toGenQueueLock(l *omsDomain.QueueLock) *genoms.QueueLock {
	if l == nil {
		return nil
	}
	return &genoms.QueueLock{Activity: genoms.QueueLockActivity(l.Activity), UserId: l.UserID, Since: l.Since}
}

// MarkOrderPicked is the picking module's list mode -- see
// appoms.MarkOrderPickedUseCase.
func (h *Handler) MarkOrderPicked(c *gin.Context, id genoms.Id) {
	if err := mapError(c, h.markOrderPicked.Execute(c.Request.Context(), id, lockActorFrom(middleware.ClaimsFromGin(c)))); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}
