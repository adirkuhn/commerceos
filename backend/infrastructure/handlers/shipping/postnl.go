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
type getPostNLConnectionUseCase interface {
	Execute(ctx context.Context, connectionID string) (*integrationsDomain.Connection, error)
}
type updatePostNLConnectionUseCase interface {
	Execute(ctx context.Context, connectionID string, in appintegrations.UpdatePostNLConnectionInput) (*integrationsDomain.Connection, error)
}

// PostNLHandler serves PostNL's own endpoints: connecting an account and
// its account/environment on the settings page. Embedded in Handler like
// PreferencesHandler.
type PostNLHandler struct {
	connect connectPostNLUseCase
	get     getPostNLConnectionUseCase
	update  updatePostNLConnectionUseCase
}

func NewPostNLHandler(connect connectPostNLUseCase, get getPostNLConnectionUseCase, update updatePostNLConnectionUseCase) *PostNLHandler {
	return &PostNLHandler{connect: connect, get: get, update: update}
}

func (h *PostNLHandler) GetPostNLConnection(c *gin.Context, connectionId genshipping.ConnectionId) {
	conn, err := h.get.Execute(c.Request.Context(), connectionId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, genshipping.PostNLConnectionResponse{Data: postnlConnectionToResponse(conn, connectionId)})
}

func (h *PostNLHandler) UpdatePostNLConnection(c *gin.Context, connectionId genshipping.ConnectionId) {
	var req genshipping.UpdatePostNLConnectionJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	conn, err := h.update.Execute(c.Request.Context(), connectionId.String(), appintegrations.UpdatePostNLConnectionInput{
		APIKey: req.ApiKey, CustomerCode: req.CustomerCode, CustomerNumber: req.CustomerNumber, Sandbox: req.Sandbox,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, genshipping.PostNLConnectionResponse{Data: postnlConnectionToResponse(conn, connectionId)})
}

func postnlConnectionToResponse(conn *integrationsDomain.Connection, id genshipping.ConnectionId) genshipping.PostNLConnection {
	return genshipping.PostNLConnection{
		ConnectionId: id, CustomerCode: conn.PostNLCustomerCode, CustomerNumber: conn.PostNLCustomerNumber, Sandbox: conn.PostNLSandbox,
	}
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
