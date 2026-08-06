package suppliers_test

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

	gensuppliers "github.com/adirkuhn/commerceos/api/suppliers"
	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	supplierhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/suppliers"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListSuppliers struct {
	out []suppliersDomain.Supplier
	err error
}

func (s *stubListSuppliers) Execute(_ context.Context) ([]suppliersDomain.Supplier, error) {
	return s.out, s.err
}

type stubCreateSupplier struct {
	out      *suppliersDomain.Supplier
	err      error
	gotInput appsuppliers.CreateSupplierInput
}

func (s *stubCreateSupplier) Execute(_ context.Context, in appsuppliers.CreateSupplierInput) (*suppliersDomain.Supplier, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &suppliersDomain.Supplier{ID: "supplier-1", Name: in.Name, Status: in.Status}, nil
}

type stubUpdateSupplier struct {
	out      *suppliersDomain.Supplier
	err      error
	gotInput appsuppliers.UpdateSupplierInput
}

func (s *stubUpdateSupplier) Execute(_ context.Context, in appsuppliers.UpdateSupplierInput) (*suppliersDomain.Supplier, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &suppliersDomain.Supplier{ID: in.ID, Name: in.Name, Status: in.Status}, nil
}

type stubDeleteSupplier struct {
	err   error
	calls []string
}

func (s *stubDeleteSupplier) Execute(_ context.Context, id string) error {
	s.calls = append(s.calls, id)
	return s.err
}

func newTestHandler(list *stubListSuppliers, create *stubCreateSupplier, update *stubUpdateSupplier, del *stubDeleteSupplier) *supplierhandler.Handler {
	return supplierhandler.NewHandler(list, create, update, del)
}

func doRequest(rg *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func newTestRouter(h *supplierhandler.Handler) *gin.Engine {
	r := gin.New()
	gensuppliers.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListSuppliersHandler_returnsSuppliers(t *testing.T) {
	h := newTestHandler(&stubListSuppliers{out: []suppliersDomain.Supplier{{ID: "supplier-1", Name: "Acme Foods"}}}, &stubCreateSupplier{}, &stubUpdateSupplier{}, &stubDeleteSupplier{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/suppliers", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Acme Foods")
}

func TestCreateSupplierHandler_success(t *testing.T) {
	create := &stubCreateSupplier{}
	h := newTestHandler(&stubListSuppliers{}, create, &stubUpdateSupplier{}, &stubDeleteSupplier{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/suppliers", map[string]any{
		"name": "Acme Foods", "email": "hi@acme.example", "status": "active", "lead_time_days": 14,
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "Acme Foods", create.gotInput.Name)
	assert.Equal(t, "hi@acme.example", create.gotInput.Email)
	require.NotNil(t, create.gotInput.LeadTimeDays)
	assert.Equal(t, 14, *create.gotInput.LeadTimeDays)
}

func TestCreateSupplierHandler_rejectsMissingName(t *testing.T) {
	h := newTestHandler(&stubListSuppliers{}, &stubCreateSupplier{}, &stubUpdateSupplier{}, &stubDeleteSupplier{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/suppliers", map[string]any{"status": "active"})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateSupplierHandler_success(t *testing.T) {
	update := &stubUpdateSupplier{}
	h := newTestHandler(&stubListSuppliers{}, &stubCreateSupplier{}, update, &stubDeleteSupplier{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/suppliers/supplier-1", map[string]any{
		"name": "Acme Foods Ltd", "status": "inactive",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "supplier-1", update.gotInput.ID)
	assert.Equal(t, "Acme Foods Ltd", update.gotInput.Name)
	assert.Equal(t, suppliersDomain.SupplierStatusInactive, update.gotInput.Status)
}

func TestUpdateSupplierHandler_notFound(t *testing.T) {
	update := &stubUpdateSupplier{err: suppliersDomain.ErrSupplierNotFound}
	h := newTestHandler(&stubListSuppliers{}, &stubCreateSupplier{}, update, &stubDeleteSupplier{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/suppliers/missing", map[string]any{
		"name": "X", "status": "active",
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSupplierHandler_success(t *testing.T) {
	del := &stubDeleteSupplier{}
	h := newTestHandler(&stubListSuppliers{}, &stubCreateSupplier{}, &stubUpdateSupplier{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/suppliers/supplier-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"supplier-1"}, del.calls)
}

func TestDeleteSupplierHandler_notFound(t *testing.T) {
	del := &stubDeleteSupplier{err: suppliersDomain.ErrSupplierNotFound}
	h := newTestHandler(&stubListSuppliers{}, &stubCreateSupplier{}, &stubUpdateSupplier{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/suppliers/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}
