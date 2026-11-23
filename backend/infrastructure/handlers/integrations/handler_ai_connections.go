package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// The tenant's ordered AI connections (see appintegrations' ai_connections).

func (h *Handler) ListAIConnections(c *gin.Context) {
	conns, err := h.aiConnections.List.Execute(c.Request.Context())
	if mapError(c, err) != nil {
		return
	}
	middleware.OK(c, toGenAIConnections(conns))
}

func (h *Handler) CreateAIConnection(c *gin.Context) {
	var req genintegrations.CreateAIConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	in := appintegrations.AIConnectionInput{
		Provider: integrationsDomain.AIProvider(req.Provider), APIKey: deref(req.ApiKey), BaseURL: deref(req.BaseUrl),
		Model: deref(req.Model), FallbackModels: deref(req.FallbackModels), Label: deref(req.Label), Enabled: req.Enabled,
	}
	conn, err := h.aiConnections.Create.Execute(c.Request.Context(), in)
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenAIConnection(*conn)})
}

func (h *Handler) UpdateAIConnection(c *gin.Context, id openapi_types.UUID) {
	var req genintegrations.UpdateAIConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	p := appintegrations.AIConnectionPatch{
		APIKey: req.ApiKey, BaseURL: req.BaseUrl, Model: req.Model, FallbackModels: req.FallbackModels,
		Label: req.Label, Enabled: req.Enabled,
	}
	if req.Provider != nil {
		provider := integrationsDomain.AIProvider(*req.Provider)
		p.Provider = &provider
	}
	conn, err := h.aiConnections.Update.Execute(c.Request.Context(), id.String(), p)
	if mapError(c, err) != nil {
		return
	}
	middleware.OK(c, toGenAIConnection(*conn))
}

func (h *Handler) DeleteAIConnection(c *gin.Context, id openapi_types.UUID) {
	if mapError(c, h.aiConnections.Delete.Execute(c.Request.Context(), id.String())) != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ReorderAIConnections(c *gin.Context) {
	var req genintegrations.ReorderAIConnectionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ids := make([]string, len(req.Ids))
	for i, id := range req.Ids {
		ids[i] = id.String()
	}
	conns, err := h.aiConnections.Reorder.Execute(c.Request.Context(), ids)
	if mapError(c, err) != nil {
		return
	}
	middleware.OK(c, toGenAIConnections(conns))
}

func toGenAIConnections(conns []integrationsDomain.AIConnection) []genintegrations.AIConnection {
	out := make([]genintegrations.AIConnection, len(conns))
	for i, conn := range conns {
		out[i] = toGenAIConnection(conn)
	}
	return out
}

// toGenAIConnection never includes the key -- only whether one is stored.
func toGenAIConnection(c integrationsDomain.AIConnection) genintegrations.AIConnection {
	resp := genintegrations.AIConnection{
		Id: c.ID, Provider: genintegrations.AIProvider(c.Provider), Position: c.Position, Label: c.Label,
		Enabled: c.Enabled, HasApiKey: c.APIKeyEncrypted != "",
		Status: genintegrations.ConnectionStatus(c.Status), CreatedAt: c.CreatedAt,
	}
	if c.BaseURL != "" {
		resp.BaseUrl = &c.BaseURL
	}
	if c.Model != "" {
		resp.Model = &c.Model
	}
	if c.FallbackModels != "" {
		resp.FallbackModels = &c.FallbackModels
	}
	return resp
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
