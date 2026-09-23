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

type listPackingQueueUC interface {
	Execute(ctx context.Context, page, perPage int) (*appoms.ListPackingQueueResult, error)
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
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	result, err := h.listPackingQueue.Execute(c.Request.Context(), page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	entries := make([]genoms.PackingQueueEntry, len(result.Entries))
	for i, e := range result.Entries {
		entries[i] = genoms.PackingQueueEntry{SalesOrder: toGenSalesOrder(e.Order), Lock: toGenQueueLock(e.Lock)}
	}
	resp := genoms.PackingQueue{Entries: entries, Total: result.Total}
	c.JSON(http.StatusOK, genoms.PackingQueueResponse{Data: &resp})
}

func toGenQueueLock(l *omsDomain.QueueLock) *genoms.PackingQueueLock {
	if l == nil {
		return nil
	}
	return &genoms.PackingQueueLock{Activity: genoms.PackingQueueLockActivity(l.Activity), UserId: l.UserID, Since: l.Since}
}
