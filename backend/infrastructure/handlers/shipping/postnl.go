package shipping

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type connectPostNLUseCase interface {
	Execute(ctx context.Context, in appintegrations.ConnectPostNLInput) (*integrationsDomain.Connection, error)
}

// PostNLHandler serves PostNL's own endpoints (connecting an account).
// Embedded in Handler like PreferencesHandler.
type PostNLHandler struct {
	connect connectPostNLUseCase
}

func NewPostNLHandler(connect connectPostNLUseCase) *PostNLHandler {
	return &PostNLHandler{connect: connect}
}

func (h *PostNLHandler) ConnectPostNL(c *gin.Context) {
	var req genshipping.ConnectPostNLJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	conn, err := h.connect.Execute(c.Request.Context(), appintegrations.ConnectPostNLInput{
		APIKey: req.ApiKey, CustomerCode: req.CustomerCode, CustomerNumber: req.CustomerNumber, Sandbox: req.Sandbox,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"connection_id": conn.ID, "provider": string(conn.Provider), "status": string(conn.Status),
	}, "error": nil})
}
