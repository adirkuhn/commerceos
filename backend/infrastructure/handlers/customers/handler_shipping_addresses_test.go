package customers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	customerhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/customers"
)

type stubShippingAddresses struct {
	book         customersDomain.ShippingBook
	err          error
	gotCustomer  string
	gotAddress   string
	gotInput     appcustomers.ShippingAddressInput
	gotAsDefault bool
}

func (s *stubShippingAddresses) List(_ context.Context, customerID string) (customersDomain.ShippingBook, error) {
	s.gotCustomer = customerID
	return s.book, s.err
}

func (s *stubShippingAddresses) Add(_ context.Context, customerID string, in appcustomers.ShippingAddressInput, asDefault bool) (*customersDomain.ShippingAddress, error) {
	s.gotCustomer, s.gotInput, s.gotAsDefault = customerID, in, asDefault
	if s.err != nil {
		return nil, s.err
	}
	return &customersDomain.ShippingAddress{ID: "sa-1", CustomerID: customerID, Label: in.Label, Address: in.Address, IsDefault: asDefault}, nil
}

func (s *stubShippingAddresses) Update(_ context.Context, customerID, id string, in appcustomers.ShippingAddressInput) (*customersDomain.ShippingAddress, error) {
	s.gotCustomer, s.gotAddress, s.gotInput = customerID, id, in
	if s.err != nil {
		return nil, s.err
	}
	return &customersDomain.ShippingAddress{ID: id, CustomerID: customerID, Label: in.Label, Address: in.Address}, nil
}

func (s *stubShippingAddresses) Remove(_ context.Context, customerID, id string) error {
	s.gotCustomer, s.gotAddress = customerID, id
	return s.err
}

func (s *stubShippingAddresses) SetDefault(_ context.Context, customerID, id string) (customersDomain.ShippingBook, error) {
	s.gotCustomer, s.gotAddress = customerID, id
	return s.book, s.err
}

func shippingRouter(s *stubShippingAddresses) *gin.Engine {
	h := customerhandler.NewHandler(&stubListCustomers{}, &stubCreateCustomer{}, &stubUpdateCustomer{}, &stubDeleteCustomer{}, &stubListCustomerWhatsAppMessages{}, &stubGetCustomer{}).
		WithShippingAddresses(customerhandler.ShippingAddressUseCases{List: s.List, Add: s.Add, Update: s.Update, Remove: s.Remove, SetDefault: s.SetDefault})
	r := gin.New()
	gencustomers.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListShippingAddresses_success(t *testing.T) {
	s := &stubShippingAddresses{book: customersDomain.ShippingBook{{ID: "sa-1", CustomerID: "c-1", Label: "Warehouse", IsDefault: true,
		Address: customersDomain.Address{Address1: "Dock 2", City: "Utrecht"}}}}
	w := doRequest(shippingRouter(s), "GET", "/api/v1/customers/c-1/shipping-addresses", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "c-1", s.gotCustomer)
	assert.JSONEq(t, `{"data":[{"id":"sa-1","customer_id":"c-1","label":"Warehouse","first_name":"","last_name":"","company":"",
		"address_1":"Dock 2","address_2":"","city":"Utrecht","state":"","postcode":"","country":"","phone":"","is_default":true}]}`, w.Body.String())
}

func TestListShippingAddresses_emptyIsAnEmptyList(t *testing.T) {
	w := doRequest(shippingRouter(&stubShippingAddresses{}), "GET", "/api/v1/customers/c-1/shipping-addresses", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":[]}`, w.Body.String())
}

func TestListShippingAddresses_unknownCustomerIs404(t *testing.T) {
	w := doRequest(shippingRouter(&stubShippingAddresses{err: customersDomain.ErrCustomerNotFound}), "GET", "/api/v1/customers/x/shipping-addresses", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddShippingAddress_success(t *testing.T) {
	s := &stubShippingAddresses{}
	w := doRequest(shippingRouter(s), "POST", "/api/v1/customers/c-1/shipping-addresses",
		map[string]any{"label": "Warehouse", "address_1": "Dock 2", "city": "Utrecht", "phone": "06", "is_default": true})

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "Warehouse", s.gotInput.Label)
	assert.Equal(t, customersDomain.Address{Address1: "Dock 2", City: "Utrecht", Phone: "06"}, s.gotInput.Address)
	assert.True(t, s.gotAsDefault)
	assert.Contains(t, w.Body.String(), `"id":"sa-1"`)
}

func TestAddShippingAddress_errors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
	}{
		{customersDomain.ErrShippingAddressLineRequired, http.StatusUnprocessableEntity},
		{customersDomain.ErrShippingLabelTooLong, http.StatusUnprocessableEntity},
		{customersDomain.ErrTooManyShippingAddresses, http.StatusUnprocessableEntity},
		{customersDomain.ErrCustomerNotFound, http.StatusNotFound},
	} {
		w := doRequest(shippingRouter(&stubShippingAddresses{err: tc.err}), "POST", "/api/v1/customers/c-1/shipping-addresses", map[string]any{"address_1": "Dock 2"})
		assert.Equal(t, tc.code, w.Code, tc.err.Error())
	}
}

func TestAddShippingAddress_badJSONIs422(t *testing.T) {
	w := doRequest(shippingRouter(&stubShippingAddresses{}), "POST", "/api/v1/customers/c-1/shipping-addresses", "not an object")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateShippingAddress_success(t *testing.T) {
	s := &stubShippingAddresses{}
	w := doRequest(shippingRouter(s), "PATCH", "/api/v1/customers/c-1/shipping-addresses/sa-1", map[string]any{"address_1": "Dock 3"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "sa-1", s.gotAddress)
	assert.Equal(t, "Dock 3", s.gotInput.Address.Address1)
}

func TestUpdateShippingAddress_notFoundIs404(t *testing.T) {
	w := doRequest(shippingRouter(&stubShippingAddresses{err: customersDomain.ErrShippingAddressNotFound}), "PATCH", "/api/v1/customers/c-1/shipping-addresses/x", map[string]any{"address_1": "Dock 3"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveShippingAddress(t *testing.T) {
	s := &stubShippingAddresses{}
	w := doRequest(shippingRouter(s), "DELETE", "/api/v1/customers/c-1/shipping-addresses/sa-1", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sa-1", s.gotAddress)

	w = doRequest(shippingRouter(&stubShippingAddresses{err: customersDomain.ErrShippingAddressNotFound}), "DELETE", "/api/v1/customers/c-1/shipping-addresses/x", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetDefaultShippingAddress(t *testing.T) {
	s := &stubShippingAddresses{book: customersDomain.ShippingBook{{ID: "sa-2", CustomerID: "c-1", IsDefault: true, Address: customersDomain.Address{Address1: "Dock 2"}}}}
	w := doRequest(shippingRouter(s), "PUT", "/api/v1/customers/c-1/shipping-addresses/sa-2/default", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sa-2", s.gotAddress)
	assert.Contains(t, w.Body.String(), `"is_default":true`)

	w = doRequest(shippingRouter(&stubShippingAddresses{err: customersDomain.ErrShippingAddressNotFound}), "PUT", "/api/v1/customers/c-1/shipping-addresses/x/default", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
