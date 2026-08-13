package customers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	customerhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/customers"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListCustomers struct {
	out []customersDomain.Customer
	err error
}

func (s *stubListCustomers) Execute(_ context.Context) ([]customersDomain.Customer, error) {
	return s.out, s.err
}

type stubCreateCustomer struct {
	out      *customersDomain.Customer
	err      error
	gotInput appcustomers.CreateCustomerInput
}

func (s *stubCreateCustomer) Execute(_ context.Context, in appcustomers.CreateCustomerInput) (*customersDomain.Customer, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &customersDomain.Customer{ID: "customer-1", Email: in.Email}, nil
}

type stubUpdateCustomer struct {
	out      *customersDomain.Customer
	err      error
	gotInput appcustomers.UpdateCustomerInput
}

func (s *stubUpdateCustomer) Execute(_ context.Context, in appcustomers.UpdateCustomerInput) (*customersDomain.Customer, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &customersDomain.Customer{ID: in.ID, Email: in.Email}, nil
}

type stubDeleteCustomer struct {
	err   error
	calls []string
}

func (s *stubDeleteCustomer) Execute(_ context.Context, id string) error {
	s.calls = append(s.calls, id)
	return s.err
}

func newTestRouter(list *stubListCustomers, create *stubCreateCustomer, update *stubUpdateCustomer, del *stubDeleteCustomer) *gin.Engine {
	h := customerhandler.NewHandler(list, create, update, del)
	r := gin.New()
	gencustomers.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListCustomers_success(t *testing.T) {
	list := &stubListCustomers{out: []customersDomain.Customer{{ID: "c1", Email: "a@b.com"}}}
	w := doRequest(newTestRouter(list, &stubCreateCustomer{}, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "GET", "/api/v1/customers", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "a@b.com")
}

func TestCreateCustomer_success(t *testing.T) {
	create := &stubCreateCustomer{}
	body := gencustomers.CreateCustomerRequest{Email: "jane@example.com"}
	w := doRequest(newTestRouter(&stubListCustomers{}, create, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "jane@example.com", create.gotInput.Email)
}

func TestCreateCustomer_rejectsMissingEmail(t *testing.T) {
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers", gencustomers.CreateCustomerRequest{})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateCustomer_roundTripsBillingAddress(t *testing.T) {
	create := &stubCreateCustomer{}
	addr1 := "123 Main St"
	body := gencustomers.CreateCustomerRequest{Email: "jane@example.com", Billing: &gencustomers.Address{Address1: &addr1}}
	doRequest(newTestRouter(&stubListCustomers{}, create, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers", body)

	assert.Equal(t, "123 Main St", create.gotInput.Billing.Address1)
}

func TestUpdateCustomer_success(t *testing.T) {
	update := &stubUpdateCustomer{}
	body := gencustomers.UpdateCustomerRequest{Email: "new@example.com"}
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, update, &stubDeleteCustomer{}), "PATCH", "/api/v1/customers/c1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "c1", update.gotInput.ID)
	assert.Equal(t, "new@example.com", update.gotInput.Email)
}

func TestUpdateCustomer_notFound(t *testing.T) {
	update := &stubUpdateCustomer{err: customersDomain.ErrCustomerNotFound}
	body := gencustomers.UpdateCustomerRequest{Email: "new@example.com"}
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, update, &stubDeleteCustomer{}), "PATCH", "/api/v1/customers/missing", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteCustomer_success(t *testing.T) {
	del := &stubDeleteCustomer{}
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, &stubUpdateCustomer{}, del), "DELETE", "/api/v1/customers/c1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"c1"}, del.calls)
}

func TestDeleteCustomer_notFound(t *testing.T) {
	del := &stubDeleteCustomer{err: customersDomain.ErrCustomerNotFound}
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, &stubUpdateCustomer{}, del), "DELETE", "/api/v1/customers/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
