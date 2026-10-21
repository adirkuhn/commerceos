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
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

type listShippingConnectionsUseCase interface {
	Execute(ctx context.Context) (*appshipping.ShippingConnections, error)
}
type setDefaultShippingConnectionUseCase interface {
	Execute(ctx context.Context, connectionID *string) error
}
type getShippingAddressesUseCase interface {
	Execute(ctx context.Context) (*appshipping.ShippingAddresses, error)
}
type updateShippingAddressesUseCase interface {
	Execute(ctx context.Context, in appshipping.UpdateShippingAddressesInput) error
}

// PreferencesHandler serves the tenant's shipping preferences: packing's
// shipping-connection picker (which connections can ship, which one is
// the default) and the label sender/return addresses. Embedded in Handler
// so it satisfies genshipping.ServerInterface's matching operations.
type PreferencesHandler struct {
	list            listShippingConnectionsUseCase
	set             setDefaultShippingConnectionUseCase
	getAddresses    getShippingAddressesUseCase
	updateAddresses updateShippingAddressesUseCase
}

func NewPreferencesHandler(list listShippingConnectionsUseCase, set setDefaultShippingConnectionUseCase, getAddresses getShippingAddressesUseCase, updateAddresses updateShippingAddressesUseCase) *PreferencesHandler {
	return &PreferencesHandler{list: list, set: set, getAddresses: getAddresses, updateAddresses: updateAddresses}
}

func (h *PreferencesHandler) GetShippingAddresses(c *gin.Context) {
	h.writeAddresses(c)
}

func (h *PreferencesHandler) UpdateShippingAddresses(c *gin.Context) {
	var req genshipping.UpdateShippingAddressesJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	in := appshipping.UpdateShippingAddressesInput{Sender: addressFromRequest(req.SenderAddress), Return: addressFromRequest(req.ReturnAddress)}
	if err := h.updateAddresses.Execute(c.Request.Context(), in); err != nil {
		mapErr(c, err)
		return
	}
	h.writeAddresses(c)
}

func (h *PreferencesHandler) writeAddresses(c *gin.Context) {
	result, err := h.getAddresses.Execute(c.Request.Context())
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, genshipping.ShippingAddressesResponse{Data: genshipping.ShippingAddresses{
		SenderAddress: addressToResponse(result.Sender), ReturnAddress: addressToResponse(result.Return),
		CompanyAddress: addressToResponse(result.CompanyAddress),
	}})
}

func addressFromRequest(a *genshipping.ShippingAddress) *shippingDomain.ShippingAddress {
	if a == nil {
		return nil
	}
	return &shippingDomain.ShippingAddress{
		Company: a.Company, Street: a.Street, Number: a.Number, NumberSuffix: deref(a.NumberSuffix),
		PostalCode: a.PostalCode, City: a.City, CC: a.Cc, Email: deref(a.Email), Phone: deref(a.Phone),
	}
}

func addressToResponse(a *shippingDomain.ShippingAddress) *genshipping.ShippingAddress {
	if a == nil {
		return nil
	}
	return &genshipping.ShippingAddress{
		Company: a.Company, Street: a.Street, Number: a.Number, NumberSuffix: optional(a.NumberSuffix),
		PostalCode: a.PostalCode, City: a.City, Cc: a.CC, Email: optional(a.Email), Phone: optional(a.Phone),
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
		missing := make([]genshipping.ShippingConnectionMissing, 0, len(conn.Missing))
		for _, m := range conn.Missing {
			missing = append(missing, genshipping.ShippingConnectionMissing(m))
		}
		out.Connections = append(out.Connections, genshipping.ShippingConnection{
			Id: id, Provider: string(conn.Provider), Status: string(conn.Status), CreatedAt: conn.CreatedAt,
			IsDefault: conn.ID == r.DefaultConnectionID, Missing: missing,
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
