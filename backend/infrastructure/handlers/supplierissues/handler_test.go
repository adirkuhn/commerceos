package supplierissues_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	gensupplierissues "github.com/adirkuhn/commerceos/api/supplierissues"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/supplierissues"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	poID   = "3f6d1c2a-0b7e-4c1d-9a2f-5e8b7c6d4a31"
	scanID = "8a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"
)

type stubList struct {
	got string
	out *appprocurement.SupplierIssueList
	err error
}

func (s *stubList) Execute(_ context.Context, id string) (*appprocurement.SupplierIssueList, error) {
	s.got = id
	return s.out, s.err
}

type stubUpdate struct {
	got    appprocurement.UpdateSupplierIssueStatusInput
	userID string
	called bool
	out    *appprocurement.SupplierIssueView
	err    error
}

func (s *stubUpdate) Execute(ctx context.Context, in appprocurement.UpdateSupplierIssueStatusInput) (*appprocurement.SupplierIssueView, error) {
	s.got, s.userID, s.called = in, tenant.UserIDFromContext(ctx), true
	return s.out, s.err
}

type stubEmail struct {
	got    appprocurement.EmailSupplierIssuesInput
	called bool
	out    *appprocurement.EmailSupplierIssuesResult
	err    error
}

func (s *stubEmail) Execute(_ context.Context, in appprocurement.EmailSupplierIssuesInput) (*appprocurement.EmailSupplierIssuesResult, error) {
	s.got, s.called = in, true
	return s.out, s.err
}

func newRouter(t *testing.T, list *stubList, update *stubUpdate, email *stubEmail) *gin.Engine {
	t.Helper()
	spec, err := gensupplierissues.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte("s")))
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	gensupplierissues.RegisterHandlers(g, handler.NewHandler(list, update, email))
	return r
}

func token(t *testing.T) string {
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "user-1", TenantID: "8d0c3f4e-1a2b-4c5d-9e8f-7a6b5c4d3e2f", Role: middleware.RoleCustomerAdmin,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("s"))
	require.NoError(t, err)
	return s
}

func do(t *testing.T, r *gin.Engine, method, path, body string, withToken bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if withToken {
		req.Header.Set("Authorization", "Bearer "+token(t))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func strPtr(s string) *string { return &s }

var occurred = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func missingIssue() appprocurement.SupplierIssueView {
	return appprocurement.SupplierIssueView{
		SupplierIssue: procurementDomain.SupplierIssue{
			Ref:  procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingShortfall, SourceID: "line-1"},
			Kind: procurementDomain.SupplierIssueMissing, LineItemID: strPtr("line-1"), SKUID: strPtr("sku-1"),
			SupplierSKU: "B-100", SupplierDescription: "Pao de queijo 1kg", ExpectedQty: 10, ReceivedQty: 7, Quantity: 3,
			OccurredAt: occurred, Status: procurementDomain.SupplierIssueStatusOpen,
		},
		SKUCode: strPtr("PDQ-1KG"), ProductName: strPtr("Pão de queijo"),
	}
}

func TestListSupplierIssues_MapsTheList(t *testing.T) {
	damaged := appprocurement.SupplierIssueView{SupplierIssue: procurementDomain.SupplierIssue{
		Ref:  procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: scanID},
		Kind: procurementDomain.SupplierIssueDamaged, Quantity: 2, Reason: "crushed", ReportedBy: "ana", OccurredAt: occurred,
		Status: procurementDomain.SupplierIssueStatusClaimed, StatusNote: "emailed", StatusUpdatedAt: &occurred,
	}}
	list := &stubList{out: &appprocurement.SupplierIssueList{
		PurchaseOrderID: poID, BookingInProgress: true,
		Supplier: &appprocurement.SupplierContact{ID: "sup-1", Name: "Bauducco NL", Email: "orders@bauducco.example"},
		Issues:   []appprocurement.SupplierIssueView{missingIssue(), damaged},
	}}
	r := newRouter(t, list, &stubUpdate{}, &stubEmail{})

	w := do(t, r, http.MethodGet, "/api/v1/procurement/purchase-orders/"+poID+"/supplier-issues", "", true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, poID, list.got)
	var body gensupplierissues.SupplierIssueListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Data.BookingInProgress)
	assert.Equal(t, &gensupplierissues.SupplierContact{Id: "sup-1", Name: "Bauducco NL", Email: "orders@bauducco.example"}, body.Data.Supplier)
	require.Len(t, body.Data.Issues, 2)
	m := body.Data.Issues[0]
	assert.Equal(t, gensupplierissues.BookingShortfall, m.Source)
	assert.Equal(t, gensupplierissues.Missing, m.Kind)
	assert.Equal(t, int64(3), m.Quantity)
	assert.Equal(t, int64(10), *m.ExpectedQty)
	assert.Equal(t, int64(7), *m.ReceivedQty)
	assert.Equal(t, "PDQ-1KG", *m.SkuCode)
	assert.Equal(t, "Pão de queijo", *m.ProductName)
	assert.Equal(t, "Pao de queijo 1kg", m.SupplierDescription)
	assert.Equal(t, gensupplierissues.Open, m.Status)
	assert.Nil(t, m.StatusUpdatedAt)
	d := body.Data.Issues[1]
	assert.Nil(t, d.ExpectedQty, "expected/received are for missing items only")
	assert.Nil(t, d.SkuCode)
	assert.Equal(t, "crushed", d.Reason)
	assert.Equal(t, gensupplierissues.Claimed, d.Status)
	assert.Equal(t, "emailed", d.StatusNote)
	assert.True(t, occurred.Equal(*d.StatusUpdatedAt))
}

func TestListSupplierIssues_EmptyListIsAnArray(t *testing.T) {
	r := newRouter(t, &stubList{out: &appprocurement.SupplierIssueList{PurchaseOrderID: poID}}, &stubUpdate{}, &stubEmail{})

	w := do(t, r, http.MethodGet, "/api/v1/procurement/purchase-orders/"+poID+"/supplier-issues", "", true)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"issues":[]`)
	assert.NotContains(t, w.Body.String(), `"supplier"`)
}

func TestListSupplierIssues_Errors(t *testing.T) {
	path := "/api/v1/procurement/purchase-orders/" + poID + "/supplier-issues"
	r := newRouter(t, &stubList{err: fmt.Errorf("listSupplierIssues: %w", procurementDomain.ErrPurchaseOrderNotFound)}, &stubUpdate{}, &stubEmail{})

	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodGet, path, "", false).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodGet, "/api/v1/procurement/purchase-orders/not-a-uuid/supplier-issues", "", true).Code)
	w := do(t, r, http.MethodGet, path, "", true)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "PURCHASE_ORDER_NOT_FOUND")

	r = newRouter(t, &stubList{err: errors.New("db down")}, &stubUpdate{}, &stubEmail{})
	w = do(t, r, http.MethodGet, path, "", true)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "db down", "internal errors never leak")
}

func TestUpdateSupplierIssueStatus_PassesTheIssueAndCaller(t *testing.T) {
	view := missingIssue()
	view.Status, view.StatusNote = procurementDomain.SupplierIssueStatusResolved, "CN-7"
	update := &stubUpdate{out: &view}
	r := newRouter(t, &stubList{}, update, &stubEmail{})

	w := do(t, r, http.MethodPatch, "/api/v1/procurement/purchase-orders/"+poID+"/supplier-issues/booking_scan/"+scanID,
		`{"status":"resolved","note":"CN-7"}`, true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, appprocurement.UpdateSupplierIssueStatusInput{
		PurchaseOrderID: poID, Ref: procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: scanID},
		Status: procurementDomain.SupplierIssueStatusResolved, Note: "CN-7",
	}, update.got)
	assert.Equal(t, "user-1", update.userID)
	var body gensupplierissues.SupplierIssueResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, gensupplierissues.Resolved, body.Data.Status)
	assert.Equal(t, "CN-7", body.Data.StatusNote)
}

func TestUpdateSupplierIssueStatus_Errors(t *testing.T) {
	path := "/api/v1/procurement/purchase-orders/" + poID + "/supplier-issues/booking_scan/" + scanID
	update := &stubUpdate{}
	r := newRouter(t, &stubList{}, update, &stubEmail{})

	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodPatch, path, `{"status":"claimed"}`, false).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPatch, path, `{"status":"lost"}`, true).Code, "contract rejects an unknown status")
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPatch, path, `{}`, true).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPatch, "/api/v1/procurement/purchase-orders/"+poID+"/supplier-issues/invoice/"+scanID, `{"status":"claimed"}`, true).Code)
	assert.False(t, update.called)

	for err, status := range map[error]int{
		procurementDomain.ErrPurchaseOrderNotFound:      http.StatusNotFound,
		procurementDomain.ErrSupplierIssueNotFound:      http.StatusNotFound,
		procurementDomain.ErrInvalidSupplierIssueStatus: http.StatusUnprocessableEntity,
		procurementDomain.ErrSupplierIssueNoteTooLong:   http.StatusUnprocessableEntity,
	} {
		r := newRouter(t, &stubList{}, &stubUpdate{err: fmt.Errorf("updateSupplierIssueStatus: %w", err)}, &stubEmail{})
		assert.Equal(t, status, do(t, r, http.MethodPatch, path, `{"status":"claimed"}`, true).Code, err.Error())
	}
}

func TestEmailSupplierIssues_PassesTheMessage(t *testing.T) {
	email := &stubEmail{out: &appprocurement.EmailSupplierIssuesResult{SentTo: "orders@bauducco.example", Claimed: 1}}
	r := newRouter(t, &stubList{}, &stubUpdate{}, email)

	w := do(t, r, http.MethodPost, "/api/v1/procurement/purchase-orders/"+poID+"/supplier-issues/email",
		`{"to":"orders@bauducco.example","subject":"Delivery problems","body":"3 x missing","issues":[{"source":"booking_scan","source_id":"`+scanID+`"}]}`, true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, appprocurement.EmailSupplierIssuesInput{
		PurchaseOrderID: poID, To: "orders@bauducco.example", Subject: "Delivery problems", Body: "3 x missing",
		Refs: []procurementDomain.SupplierIssueRef{{Source: procurementDomain.SupplierIssueSourceBookingScan, SourceID: scanID}},
	}, email.got)
	assert.JSONEq(t, `{"data":{"sent_to":"orders@bauducco.example","claimed":1}}`, w.Body.String())
}

func TestEmailSupplierIssues_Errors(t *testing.T) {
	path := "/api/v1/procurement/purchase-orders/" + poID + "/supplier-issues/email"
	valid := `{"to":"orders@bauducco.example","subject":"s","body":"b","issues":[{"source":"booking_scan","source_id":"` + scanID + `"}]}`
	email := &stubEmail{}
	r := newRouter(t, &stubList{}, &stubUpdate{}, email)

	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodPost, path, valid, false).Code)
	assert.Equal(t, http.StatusUnprocessableEntity,
		do(t, r, http.MethodPost, path, `{"to":"nope","subject":"s","body":"b","issues":[{"source":"booking_scan","source_id":"`+scanID+`"}]}`, true).Code,
		"bad address is rejected when the body binds")
	for name, body := range map[string]string{
		"no issues":   `{"to":"a@b.example","subject":"s","body":"b","issues":[]}`,
		"no subject":  `{"to":"a@b.example","subject":"","body":"b","issues":[{"source":"booking_scan","source_id":"` + scanID + `"}]}`,
		"extra field": `{"to":"a@b.example","subject":"s","body":"b","html":"<b>","issues":[{"source":"booking_scan","source_id":"` + scanID + `"}]}`,
	} {
		assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPost, path, body, true).Code, name)
	}
	assert.False(t, email.called)

	for err, status := range map[error]int{
		procurementDomain.ErrPurchaseOrderNotFound:                                                             http.StatusNotFound,
		procurementDomain.ErrSupplierIssueNotFound:                                                             http.StatusNotFound,
		procurementDomain.ErrInvalidSupplierEmail:                                                              http.StatusUnprocessableEntity,
		procurementDomain.ErrSupplierEmailIncomplete:                                                           http.StatusUnprocessableEntity,
		procurementDomain.ErrNoSupplierIssuesSelected:                                                          http.StatusUnprocessableEntity,
		fmt.Errorf("%w: %w", procurementDomain.ErrSupplierEmailNotSent, errors.New("550 mailbox unavailable")): http.StatusBadGateway,
	} {
		r := newRouter(t, &stubList{}, &stubUpdate{}, &stubEmail{err: fmt.Errorf("emailSupplierIssues: %w", err)})
		w := do(t, r, http.MethodPost, path, valid, true)
		assert.Equal(t, status, w.Code, err.Error())
	}

	r = newRouter(t, &stubList{}, &stubUpdate{}, &stubEmail{err: fmt.Errorf("emailSupplierIssues: %w: %w", procurementDomain.ErrSupplierEmailNotSent, errors.New("550 mailbox unavailable"))})
	w := do(t, r, http.MethodPost, path, valid, true)
	assert.Contains(t, w.Body.String(), "SUPPLIER_EMAIL_NOT_SENT")
	assert.Contains(t, w.Body.String(), "550 mailbox unavailable", "the mail server's reason helps fix the address")
}

// The supplier issue routes share /procurement/purchase-orders/{id}/...
// with api/procurement; gin panics at startup when the two disagree on a
// wildcard.
func TestRoutesCoexistWithTheProcurementAPI(t *testing.T) {
	r := gin.New()
	g := r.Group("/api/v1")
	assert.NotPanics(t, func() {
		genprocurement.RegisterHandlers(g, struct{ genprocurement.ServerInterface }{})
		gensupplierissues.RegisterHandlers(g, handler.NewHandler(&stubList{}, &stubUpdate{}, &stubEmail{}))
	})
}
