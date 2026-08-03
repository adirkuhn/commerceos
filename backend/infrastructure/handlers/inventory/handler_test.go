package inventory_test

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

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	inventoryhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/inventory"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListWarehouses struct {
	out []invDomain.Warehouse
	err error
}

func (s *stubListWarehouses) Execute(_ context.Context) ([]invDomain.Warehouse, error) {
	return s.out, s.err
}

type stubCreateWarehouse struct {
	out      *invDomain.Warehouse
	err      error
	gotInput appinventory.CreateWarehouseInput
}

func (s *stubCreateWarehouse) Execute(_ context.Context, in appinventory.CreateWarehouseInput) (*invDomain.Warehouse, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.Warehouse{ID: "w1", Name: in.Name}, nil
}

type stubUpdateWarehouse struct {
	out      *invDomain.Warehouse
	err      error
	gotInput appinventory.UpdateWarehouseInput
}

func (s *stubUpdateWarehouse) Execute(_ context.Context, in appinventory.UpdateWarehouseInput) (*invDomain.Warehouse, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.Warehouse{ID: in.ID, Name: in.Name}, nil
}

type stubDeleteWarehouse struct{ err error }

func (s *stubDeleteWarehouse) Execute(_ context.Context, _ string) error { return s.err }

func newTestHandler(list *stubListWarehouses, create *stubCreateWarehouse, update *stubUpdateWarehouse, del *stubDeleteWarehouse) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(list, create, update, del)
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

func newTestRouter(h *inventoryhandler.Handler) *gin.Engine {
	r := gin.New()
	geninventory.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListWarehousesHandler_returnsWarehouses(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{out: []invDomain.Warehouse{{ID: "w1", Name: "Main DC"}}}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/warehouses", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Main DC")
}

func TestCreateWarehouseHandler_success(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/warehouses", map[string]string{"name": "Main DC"})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Main DC")
}

func TestCreateWarehouseHandler_passesAddressFieldsAndDefaultsActiveTrue(t *testing.T) {
	create := &stubCreateWarehouse{}
	h := newTestHandler(&stubListWarehouses{}, create, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	body := map[string]any{"name": "Main DC", "address1": "1 Main St", "city": "Springfield", "country": "US"}
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/warehouses", body)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "1 Main St", create.gotInput.Address1)
	require.Equal(t, "Springfield", create.gotInput.City)
	require.Equal(t, "US", create.gotInput.Country)
	require.True(t, create.gotInput.Active)
}

func TestCreateWarehouseHandler_rejectsMissingName(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/warehouses", map[string]string{})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateWarehouseHandler_success(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/warehouses/w1", map[string]string{"name": "Renamed DC"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Renamed DC")
}

func TestUpdateWarehouseHandler_passesAddressFieldsAndActive(t *testing.T) {
	update := &stubUpdateWarehouse{}
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, update, &stubDeleteWarehouse{})
	body := map[string]any{"name": "Main DC", "province": "IL", "zip": "62701", "phone": "+1 555-0100", "active": false}
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/warehouses/w1", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "IL", update.gotInput.Province)
	require.Equal(t, "62701", update.gotInput.Zip)
	require.Equal(t, "+1 555-0100", update.gotInput.Phone)
	require.False(t, update.gotInput.Active)
}

func TestUpdateWarehouseHandler_notFound(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{err: invDomain.ErrWarehouseNotFound}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/warehouses/missing", map[string]string{"name": "X"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteWarehouseHandler_success(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/inventory/warehouses/w1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteWarehouseHandler_notFound(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{err: invDomain.ErrWarehouseNotFound})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/inventory/warehouses/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
