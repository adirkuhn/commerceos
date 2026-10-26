package customers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestCreateCustomer_aCompanyWithoutPersonOrEmail(t *testing.T) {
	create := &stubCreateCustomer{out: &customersDomain.Customer{ID: "c-1", Type: customersDomain.TypeCompany, CompanyName: "Sabor B.V.", ClientNumber: "3286"}}
	body := map[string]any{"customer_type": "company", "company_name": "Sabor B.V.", "kvk_number": "34073293", "vat_number": "NL001B01", "client_number": "3286"}
	w := doRequest(newTestRouter(&stubListCustomers{}, create, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers", body)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, customersDomain.TypeCompany, create.gotInput.Type)
	assert.Equal(t, "Sabor B.V.", create.gotInput.CompanyName)
	assert.Equal(t, "34073293", create.gotInput.KVKNumber)
	assert.Equal(t, "NL001B01", create.gotInput.VATNumber)
	assert.Equal(t, "3286", create.gotInput.ClientNumber)
	assert.Contains(t, w.Body.String(), `"display_name":"Sabor B.V."`)
	assert.Contains(t, w.Body.String(), `"customer_type":"company"`)
	assert.Contains(t, w.Body.String(), `"client_number":"3286"`)
}

func TestCreateCustomer_anUnknownTypeIs422(t *testing.T) {
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers",
		map[string]any{"customer_type": "reseller", "first_name": "Ana"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateCustomer_companyNameRequiredIs422(t *testing.T) {
	create := &stubCreateCustomer{err: customersDomain.ErrCompanyNameRequired}
	w := doRequest(newTestRouter(&stubListCustomers{}, create, &stubUpdateCustomer{}, &stubDeleteCustomer{}), "POST", "/api/v1/customers", map[string]any{"customer_type": "company"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "CUSTOMER_COMPANY_NAME_REQUIRED")
}

func TestUpdateCustomer_duplicateClientNumberIs409(t *testing.T) {
	update := &stubUpdateCustomer{err: customersDomain.ErrDuplicateClientNumber}
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, update, &stubDeleteCustomer{}), "PATCH", "/api/v1/customers/c-1", map[string]any{"client_number": "1"})

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "CUSTOMER_DUPLICATE_CLIENT_NUMBER")
}

func TestUpdateCustomer_leavesCompanyFieldsNotSentUnset(t *testing.T) {
	update := &stubUpdateCustomer{}
	w := doRequest(newTestRouter(&stubListCustomers{}, &stubCreateCustomer{}, update, &stubDeleteCustomer{}), "PATCH", "/api/v1/customers/c-1", map[string]any{"email": "a@x.nl"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, update.gotInput.Type)
	assert.Nil(t, update.gotInput.CompanyName)
	assert.Nil(t, update.gotInput.ClientNumber)
}
