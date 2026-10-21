package shipping

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
)

type listShippingConnectionsUseCase interface {
	Execute(ctx context.Context) (*appshipping.ShippingConnections, error)
}
type setDefaultShippingConnectionUseCase interface {
	Execute(ctx context.Context, connectionID *string) error
}

// PreferencesHandler serves packing's shipping-connection picker: which
// connections can ship and which one is the default. Embedded in Handler
// so it satisfies genshipping.ServerInterface's two matching operations.
type PreferencesHandler struct {
	list listShippingConnectionsUseCase
	set  setDefaultShippingConnectionUseCase
}

func NewPreferencesHandler(list listShippingConnectionsUseCase, set setDefaultShippingConnectionUseCase) *PreferencesHandler {
	return &PreferencesHandler{list: list, set: set}
}

func (h *PreferencesHandler) ListShippingConnections(c *gin.Context) {
	h.writeConnections(c)
}

func (h *PreferencesHandler) UpdateShippingPreferences(c *gin.Context) {
	var req genshipping.UpdateShippingPreferencesJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	var id *string
	if req.DefaultConnectionId != nil {
		s := req.DefaultConnectionId.String()
		id = &s
	}
	if err := h.set.Execute(c.Request.Context(), id); err != nil {
		mapErr(c, err)
		return
	}
	h.writeConnections(c)
}

func (h *PreferencesHandler) writeConnections(c *gin.Context) {
	result, err := h.list.Execute(c.Request.Context())
	if err != nil {
		mapErr(c, err)
		return
	}
	body, err := shippingConnectionsToResponse(result)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, genshipping.ShippingConnectionsResponse{Data: body})
}

func shippingConnectionsToResponse(r *appshipping.ShippingConnections) (genshipping.ShippingConnections, error) {
	out := genshipping.ShippingConnections{Connections: make([]genshipping.ShippingConnection, 0, len(r.Connections))}
	for _, conn := range r.Connections {
		id, err := uuid.Parse(conn.ID)
		if err != nil {
			return out, fmt.Errorf("connection id %q: %w", conn.ID, err)
		}
		out.Connections = append(out.Connections, genshipping.ShippingConnection{
			Id: id, Provider: string(conn.Provider), Status: string(conn.Status), CreatedAt: conn.CreatedAt,
			IsDefault: conn.ID == r.DefaultConnectionID,
		})
	}
	var err error
	if out.DefaultConnectionId, err = optionalUUID(r.DefaultConnectionID); err != nil {
		return out, err
	}
	if r.ConfiguredConnectionID != nil {
		if out.ConfiguredConnectionId, err = optionalUUID(*r.ConfiguredConnectionID); err != nil {
			return out, err
		}
	}
	return out, nil
}

func optionalUUID(s string) (*openapi_types.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("connection id %q: %w", s, err)
	}
	return &id, nil
}
