package tax_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gentax "github.com/adirkuhn/commerceos/api/tax"
	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	taxhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/tax"
)

func init() { gin.SetMode(gin.TestMode) }

type stubCreateTaxClass struct {
	out      *taxDomain.TaxClass
	err      error
	gotInput apptax.CreateTaxClassInput
}

func (s *stubCreateTaxClass) Execute(_ context.Context, in apptax.CreateTaxClassInput) (*taxDomain.TaxClass, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &taxDomain.TaxClass{ID: "class-1", Name: in.Name, IsDefault: in.IsDefault}, nil
}

type stubUpdateTaxClass struct {
	out      *taxDomain.TaxClass
	err      error
	gotInput apptax.UpdateTaxClassInput
}

func (s *stubUpdateTaxClass) Execute(_ context.Context, in apptax.UpdateTaxClassInput) (*taxDomain.TaxClass, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &taxDomain.TaxClass{ID: in.ID, Name: in.Name, IsDefault: in.IsDefault}, nil
}

type stubDeleteTaxClass struct {
	err   error
	gotID string
}

func (s *stubDeleteTaxClass) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubListTaxClasses struct {
	out []taxDomain.TaxClass
	err error
}

func (s *stubListTaxClasses) Execute(_ context.Context) ([]taxDomain.TaxClass, error) {
	return s.out, s.err
}

type stubUpsertTaxRate struct {
	out      *taxDomain.TaxRate
	err      error
	gotInput apptax.UpsertTaxRateInput
}

func (s *stubUpsertTaxRate) Execute(_ context.Context, in apptax.UpsertTaxRateInput) (*taxDomain.TaxRate, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &taxDomain.TaxRate{ID: "rate-1", TaxClassID: in.TaxClassID, Country: in.Country, RateBasisPoints: in.RateBasisPoints}, nil
}

type stubDeleteTaxRate struct {
	err   error
	gotID string
}

func (s *stubDeleteTaxRate) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubListTaxRates struct {
	out []taxDomain.TaxRate
	err error
}

func (s *stubListTaxRates) Execute(_ context.Context) ([]taxDomain.TaxRate, error) {
	return s.out, s.err
}

type stubUpsertExternalTaxClassLink struct {
	out      *taxDomain.ExternalTaxClassLink
	err      error
	gotInput apptax.UpsertExternalTaxClassLinkInput
}

func (s *stubUpsertExternalTaxClassLink) Execute(_ context.Context, in apptax.UpsertExternalTaxClassLinkInput) (*taxDomain.ExternalTaxClassLink, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &taxDomain.ExternalTaxClassLink{ID: "link-1", Provider: in.Provider, ExternalSlug: in.ExternalSlug, TaxClassID: in.TaxClassID}, nil
}

type stubDeleteExternalTaxClassLink struct {
	err   error
	gotID string
}

func (s *stubDeleteExternalTaxClassLink) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubListExternalTaxClassLinks struct {
	out []taxDomain.ExternalTaxClassLink
	err error
}

func (s *stubListExternalTaxClassLinks) Execute(_ context.Context) ([]taxDomain.ExternalTaxClassLink, error) {
	return s.out, s.err
}

func newTestHandler(
	create *stubCreateTaxClass, update *stubUpdateTaxClass, deleteClass *stubDeleteTaxClass, list *stubListTaxClasses,
	upsertRate *stubUpsertTaxRate, deleteRate *stubDeleteTaxRate, listRates *stubListTaxRates,
) *taxhandler.Handler {
	return newTestHandlerWithLinks(create, update, deleteClass, list, upsertRate, deleteRate, listRates,
		&stubUpsertExternalTaxClassLink{}, &stubDeleteExternalTaxClassLink{}, &stubListExternalTaxClassLinks{})
}

func newTestHandlerWithLinks(
	create *stubCreateTaxClass, update *stubUpdateTaxClass, deleteClass *stubDeleteTaxClass, list *stubListTaxClasses,
	upsertRate *stubUpsertTaxRate, deleteRate *stubDeleteTaxRate, listRates *stubListTaxRates,
	upsertLink *stubUpsertExternalTaxClassLink, deleteLink *stubDeleteExternalTaxClassLink, listLinks *stubListExternalTaxClassLinks,
) *taxhandler.Handler {
	return taxhandler.NewHandler(create, update, deleteClass, list, upsertRate, deleteRate, listRates, upsertLink, deleteLink, listLinks)
}

func newTestRouter(h *taxhandler.Handler) *gin.Engine {
	r := gin.New()
	gentax.RegisterHandlers(r.Group("/api/v1"), h)
	return r
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

func TestListTaxClassesHandler_returnsTheList(t *testing.T) {
	list := &stubListTaxClasses{out: []taxDomain.TaxClass{{ID: "class-1", Name: "Standard", IsDefault: true, UpdatedAt: time.Now()}}}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, list, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/tax/classes", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Standard")
}

func TestCreateTaxClassHandler_success(t *testing.T) {
	create := &stubCreateTaxClass{}
	h := newTestHandler(create, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/tax/classes", map[string]any{"name": "Reduced rate", "is_default": false})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Reduced rate", create.gotInput.Name)
	assert.Contains(t, w.Body.String(), "Reduced rate")
}

func TestCreateTaxClassHandler_validationError(t *testing.T) {
	create := &stubCreateTaxClass{err: taxDomain.ErrTaxClassNameRequired}
	h := newTestHandler(create, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/tax/classes", map[string]any{"name": ""})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateTaxClassHandler_success(t *testing.T) {
	update := &stubUpdateTaxClass{}
	h := newTestHandler(&stubCreateTaxClass{}, update, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/tax/classes/class-1", map[string]any{"name": "Standard rate", "is_default": true})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "class-1", update.gotInput.ID)
	assert.True(t, update.gotInput.IsDefault)
}

func TestUpdateTaxClassHandler_notFound(t *testing.T) {
	update := &stubUpdateTaxClass{err: taxDomain.ErrTaxClassNotFound}
	h := newTestHandler(&stubCreateTaxClass{}, update, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/tax/classes/missing", map[string]any{"name": "X"})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteTaxClassHandler_success(t *testing.T) {
	deleteClass := &stubDeleteTaxClass{}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, deleteClass, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/tax/classes/class-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "class-1", deleteClass.gotID)
}

func TestDeleteTaxClassHandler_inUse(t *testing.T) {
	deleteClass := &stubDeleteTaxClass{err: taxDomain.ErrTaxClassInUse}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, deleteClass, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/tax/classes/class-1", nil)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListTaxRatesHandler_returnsTheList(t *testing.T) {
	list := &stubListTaxRates{out: []taxDomain.TaxRate{{ID: "rate-1", Country: "NL", RateBasisPoints: 2100, UpdatedAt: time.Now()}}}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, list)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/tax/rates", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "NL")
}

func TestUpsertTaxRateHandler_success(t *testing.T) {
	upsert := &stubUpsertTaxRate{}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, upsert, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/tax/rates", map[string]any{
		"tax_class_id": "class-1", "country": "NL", "rate_basis_points": 2100,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "class-1", upsert.gotInput.TaxClassID)
	assert.Equal(t, "NL", upsert.gotInput.Country)
	assert.Equal(t, int32(2100), upsert.gotInput.RateBasisPoints)
}

func TestUpsertTaxRateHandler_taxClassNotFound(t *testing.T) {
	upsert := &stubUpsertTaxRate{err: taxDomain.ErrTaxClassNotFound}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, upsert, &stubDeleteTaxRate{}, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/tax/rates", map[string]any{
		"tax_class_id": "missing", "country": "NL", "rate_basis_points": 2100,
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteTaxRateHandler_success(t *testing.T) {
	deleteRate := &stubDeleteTaxRate{}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, deleteRate, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/tax/rates/rate-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rate-1", deleteRate.gotID)
}

func TestDeleteTaxRateHandler_notFound(t *testing.T) {
	deleteRate := &stubDeleteTaxRate{err: taxDomain.ErrTaxRateNotFound}
	h := newTestHandler(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, deleteRate, &stubListTaxRates{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/tax/rates/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListExternalTaxClassLinksHandler_returnsTheList(t *testing.T) {
	list := &stubListExternalTaxClassLinks{out: []taxDomain.ExternalTaxClassLink{{ID: "link-1", Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "class-1"}}}
	h := newTestHandlerWithLinks(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{},
		&stubUpsertExternalTaxClassLink{}, &stubDeleteExternalTaxClassLink{}, list)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/tax/external-links", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "reduced-rate")
}

func TestUpsertExternalTaxClassLinkHandler_success(t *testing.T) {
	upsert := &stubUpsertExternalTaxClassLink{}
	h := newTestHandlerWithLinks(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{},
		upsert, &stubDeleteExternalTaxClassLink{}, &stubListExternalTaxClassLinks{})
	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/tax/external-links", map[string]any{
		"provider": "woocommerce", "external_slug": "reduced-rate", "tax_class_id": "class-1",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "woocommerce", upsert.gotInput.Provider)
	assert.Equal(t, "class-1", upsert.gotInput.TaxClassID)
}

func TestUpsertExternalTaxClassLinkHandler_taxClassNotFound(t *testing.T) {
	upsert := &stubUpsertExternalTaxClassLink{err: taxDomain.ErrTaxClassNotFound}
	h := newTestHandlerWithLinks(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{},
		upsert, &stubDeleteExternalTaxClassLink{}, &stubListExternalTaxClassLinks{})
	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/tax/external-links", map[string]any{
		"provider": "woocommerce", "external_slug": "reduced-rate", "tax_class_id": "missing",
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteExternalTaxClassLinkHandler_success(t *testing.T) {
	deleteLink := &stubDeleteExternalTaxClassLink{}
	h := newTestHandlerWithLinks(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{},
		&stubUpsertExternalTaxClassLink{}, deleteLink, &stubListExternalTaxClassLinks{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/tax/external-links/link-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "link-1", deleteLink.gotID)
}

func TestDeleteExternalTaxClassLinkHandler_notFound(t *testing.T) {
	deleteLink := &stubDeleteExternalTaxClassLink{err: taxDomain.ErrExternalTaxClassLinkNotFound}
	h := newTestHandlerWithLinks(&stubCreateTaxClass{}, &stubUpdateTaxClass{}, &stubDeleteTaxClass{}, &stubListTaxClasses{}, &stubUpsertTaxRate{}, &stubDeleteTaxRate{}, &stubListTaxRates{},
		&stubUpsertExternalTaxClassLink{}, deleteLink, &stubListExternalTaxClassLinks{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/tax/external-links/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}
