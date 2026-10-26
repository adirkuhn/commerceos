package customers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	customerhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/customers"
)

type stubContactPersons struct {
	out         []customersDomain.ContactPerson
	err         error
	gotCustomer string
	gotPerson   string
	gotInput    appcustomers.ContactPersonInput
}

func (s *stubContactPersons) List(_ context.Context, customerID string) ([]customersDomain.ContactPerson, error) {
	s.gotCustomer = customerID
	return s.out, s.err
}

func (s *stubContactPersons) Add(_ context.Context, customerID string, in appcustomers.ContactPersonInput) (*customersDomain.ContactPerson, error) {
	s.gotCustomer, s.gotInput = customerID, in
	if s.err != nil {
		return nil, s.err
	}
	return &customersDomain.ContactPerson{ID: "cp-1", CustomerID: customerID, FirstName: in.FirstName, LastName: in.LastName, CreatedAt: time.Now()}, nil
}

func (s *stubContactPersons) Update(_ context.Context, customerID, id string, in appcustomers.ContactPersonInput) (*customersDomain.ContactPerson, error) {
	s.gotCustomer, s.gotPerson, s.gotInput = customerID, id, in
	if s.err != nil {
		return nil, s.err
	}
	return &customersDomain.ContactPerson{ID: id, CustomerID: customerID, FirstName: in.FirstName, Phone: in.Phone}, nil
}

func (s *stubContactPersons) Remove(_ context.Context, customerID, id string) error {
	s.gotCustomer, s.gotPerson = customerID, id
	return s.err
}

func contactPersonsRouter(s *stubContactPersons) *gin.Engine {
	h := customerhandler.NewHandler(&stubListCustomers{}, &stubCreateCustomer{}, &stubUpdateCustomer{}, &stubDeleteCustomer{}, &stubListCustomerWhatsAppMessages{}, &stubGetCustomer{}).
		WithContactPersons(customerhandler.ContactPersonUseCases{List: s.List, Add: s.Add, Update: s.Update, Remove: s.Remove})
	r := gin.New()
	gencustomers.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListContactPersons_success(t *testing.T) {
	s := &stubContactPersons{out: []customersDomain.ContactPerson{{ID: "cp-1", CustomerID: "c-1", FirstName: "Kylian", Role: "Employee"}}}
	w := doRequest(contactPersonsRouter(s), "GET", "/api/v1/customers/c-1/contact-persons", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "c-1", s.gotCustomer)
	assert.Contains(t, w.Body.String(), `"first_name":"Kylian"`)
	assert.Contains(t, w.Body.String(), `"role":"Employee"`)
}

func TestListContactPersons_unknownCustomerIs404(t *testing.T) {
	w := doRequest(contactPersonsRouter(&stubContactPersons{err: customersDomain.ErrCustomerNotFound}), "GET", "/api/v1/customers/x/contact-persons", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddContactPerson_success(t *testing.T) {
	s := &stubContactPersons{}
	w := doRequest(contactPersonsRouter(s), "POST", "/api/v1/customers/c-1/contact-persons", map[string]any{"first_name": "Kylian", "last_name": "Klockenbrink", "role": "Employee"})

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "Klockenbrink", s.gotInput.LastName)
	assert.Equal(t, "Employee", s.gotInput.Role)
	assert.Contains(t, w.Body.String(), `"id":"cp-1"`)
}

func TestAddContactPerson_nameRequiredIs422(t *testing.T) {
	w := doRequest(contactPersonsRouter(&stubContactPersons{err: customersDomain.ErrContactPersonNameRequired}), "POST", "/api/v1/customers/c-1/contact-persons", map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "CONTACT_PERSON_NAME_REQUIRED")
}

func TestUpdateContactPerson_success(t *testing.T) {
	s := &stubContactPersons{}
	w := doRequest(contactPersonsRouter(s), "PATCH", "/api/v1/customers/c-1/contact-persons/cp-1", map[string]any{"first_name": "Kylian", "phone": "0612"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cp-1", s.gotPerson)
	assert.Equal(t, "0612", s.gotInput.Phone)
}

func TestUpdateContactPerson_notFoundIs404(t *testing.T) {
	w := doRequest(contactPersonsRouter(&stubContactPersons{err: customersDomain.ErrContactPersonNotFound}), "PATCH", "/api/v1/customers/c-1/contact-persons/x", map[string]any{"first_name": "A"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveContactPerson_success(t *testing.T) {
	s := &stubContactPersons{}
	w := doRequest(contactPersonsRouter(s), "DELETE", "/api/v1/customers/c-1/contact-persons/cp-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cp-1", s.gotPerson)
}

func TestRemoveContactPerson_notFoundIs404(t *testing.T) {
	w := doRequest(contactPersonsRouter(&stubContactPersons{err: customersDomain.ErrContactPersonNotFound}), "DELETE", "/api/v1/customers/c-1/contact-persons/x", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateCustomer_passesItsContactPersons(t *testing.T) {
	create := &stubCreateCustomer{}
	w := doRequest(newTestRouter(&stubListCustomers{}, create, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers",
		map[string]any{"customer_type": "company", "company_name": "Claus", "contact_persons": []map[string]any{{"first_name": "Kylian", "email": "k@claus.nl"}}})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Len(t, create.gotInput.ContactPersons, 1)
	assert.Equal(t, "Kylian", create.gotInput.ContactPersons[0].FirstName)
	assert.Equal(t, "k@claus.nl", create.gotInput.ContactPersons[0].Email)
}
