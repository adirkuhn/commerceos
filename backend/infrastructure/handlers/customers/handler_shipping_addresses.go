package customers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ShippingAddressUseCases are the shipping address use cases' Execute
// methods.
type ShippingAddressUseCases struct {
	List       func(ctx context.Context, customerID string) (customersDomain.ShippingBook, error)
	Add        func(ctx context.Context, customerID string, in appcustomers.ShippingAddressInput, asDefault bool) (*customersDomain.ShippingAddress, error)
	Update     func(ctx context.Context, customerID, id string, in appcustomers.ShippingAddressInput) (*customersDomain.ShippingAddress, error)
	Remove     func(ctx context.Context, customerID, id string) error
	SetDefault func(ctx context.Context, customerID, id string) (customersDomain.ShippingBook, error)
}

// WithShippingAddresses mounts the shipping address routes' use cases.
func (h *Handler) WithShippingAddresses(uc ShippingAddressUseCases) *Handler {
	h.shippingAddresses = uc
	return h
}

func (h *Handler) ListShippingAddresses(c *gin.Context, id gencustomers.Id) {
	book, err := h.shippingAddresses.List(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, toGenShippingBook(book))
}

func (h *Handler) AddShippingAddress(c *gin.Context, id gencustomers.Id) {
	var req gencustomers.ShippingAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	a, err := h.shippingAddresses.Add(c.Request.Context(), id, toShippingAddressInput(req), boolPtrOr(req.IsDefault))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenShippingAddress(*a)
	c.JSON(http.StatusCreated, gencustomers.ShippingAddressResponse{Data: &resp})
}

func (h *Handler) UpdateShippingAddress(c *gin.Context, id gencustomers.Id, addressID gencustomers.AddressId) {
	var req gencustomers.ShippingAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	a, err := h.shippingAddresses.Update(c.Request.Context(), id, addressID, toShippingAddressInput(req))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenShippingAddress(*a)
	c.JSON(http.StatusOK, gencustomers.ShippingAddressResponse{Data: &resp})
}

func (h *Handler) RemoveShippingAddress(c *gin.Context, id gencustomers.Id, addressID gencustomers.AddressId) {
	if err := mapError(c, h.shippingAddresses.Remove(c.Request.Context(), id, addressID)); err != nil {
		return
	}
	c.JSON(http.StatusOK, gencustomers.EmptyResponse{})
}

func (h *Handler) SetDefaultShippingAddress(c *gin.Context, id gencustomers.Id, addressID gencustomers.AddressId) {
	book, err := h.shippingAddresses.SetDefault(c.Request.Context(), id, addressID)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, toGenShippingBook(book))
}

func toShippingAddressInput(req gencustomers.ShippingAddressRequest) appcustomers.ShippingAddressInput {
	return appcustomers.ShippingAddressInput{
		Label: strPtrOr(req.Label),
		Address: customersDomain.Address{
			FirstName: strPtrOr(req.FirstName), LastName: strPtrOr(req.LastName), Company: strPtrOr(req.Company),
			Address1: req.Address1, Address2: strPtrOr(req.Address2), City: strPtrOr(req.City), State: strPtrOr(req.State),
			Postcode: strPtrOr(req.Postcode), Country: strPtrOr(req.Country), Phone: strPtrOr(req.Phone),
		},
	}
}

func toGenShippingAddress(a customersDomain.ShippingAddress) gencustomers.ShippingAddress {
	ad := a.Address
	return gencustomers.ShippingAddress{
		Id: a.ID, CustomerId: a.CustomerID, Label: a.Label, IsDefault: a.IsDefault,
		FirstName: ad.FirstName, LastName: ad.LastName, Company: ad.Company, Address1: ad.Address1, Address2: ad.Address2,
		City: ad.City, State: ad.State, Postcode: ad.Postcode, Country: ad.Country, Phone: ad.Phone,
	}
}

func toGenShippingBook(book customersDomain.ShippingBook) gencustomers.ShippingAddressListResponse {
	out := make([]gencustomers.ShippingAddress, len(book))
	for i, a := range book {
		out[i] = toGenShippingAddress(a)
	}
	return gencustomers.ShippingAddressListResponse{Data: &out}
}
