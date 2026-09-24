// Session lock endpoints (release / take-over) for picking and packing, in
// their own file like packing_handler.go.
package oms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type releaseSessionLockUC interface {
	Execute(ctx context.Context, sessionID, userID string) error
}
type takeOverPackingSessionUC interface {
	Execute(ctx context.Context, sessionID string, actor omsDomain.LockActor) (*omsDomain.PackingSession, error)
}
type takeOverPickingSessionUC interface {
	Execute(ctx context.Context, sessionID string, actor omsDomain.LockActor) (*omsDomain.PickingSession, error)
}

// SessionLockUseCases groups the lock actions a picking/packing session
// offers beyond its own writes -- releasing on leave and explicit takeover.
type SessionLockUseCases struct {
	ReleasePacking  releaseSessionLockUC
	ReleasePicking  releaseSessionLockUC
	TakeOverPacking takeOverPackingSessionUC
	TakeOverPicking takeOverPickingSessionUC
}

func callerID(c *gin.Context) string {
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		return claims.UserID
	}
	return ""
}

func (h *Handler) ReleasePackingSessionLock(c *gin.Context, id genoms.Id) {
	if err := mapError(c, h.locks.ReleasePacking.Execute(c.Request.Context(), id, callerID(c))); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReleasePickingSessionLock(c *gin.Context, id genoms.Id) {
	if err := mapError(c, h.locks.ReleasePicking.Execute(c.Request.Context(), id, callerID(c))); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) TakeOverPackingSession(c *gin.Context, id genoms.Id) {
	session, err := h.locks.TakeOverPacking.Execute(c.Request.Context(), id, lockActorFrom(middleware.ClaimsFromGin(c)))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPackingSession(*session)
	c.JSON(http.StatusOK, genoms.PackingSessionResponse{Data: &resp})
}

func (h *Handler) TakeOverPickingSession(c *gin.Context, id genoms.Id) {
	session, err := h.locks.TakeOverPicking.Execute(c.Request.Context(), id, lockActorFrom(middleware.ClaimsFromGin(c)))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPickingSession(*session)
	c.JSON(http.StatusOK, genoms.PickingSessionResponse{Data: &resp})
}
