package search_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gensearch "github.com/adirkuhn/commerceos/api/search"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	searchhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/search"
)

func init() { gin.SetMode(gin.TestMode) }

type stubGlobalSearch struct {
	gotQuery string
	out      *appsearch.GlobalSearchResult
	err      error
}

func (s *stubGlobalSearch) Execute(_ context.Context, query string) (*appsearch.GlobalSearchResult, error) {
	s.gotQuery = query
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appsearch.GlobalSearchResult{}, nil
}

func newTestRouter(h *searchhandler.Handler) *gin.Engine {
	r := gin.New()
	gensearch.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestGlobalSearchHandler_returnsGroupedResults(t *testing.T) {
	stub := &stubGlobalSearch{out: &appsearch.GlobalSearchResult{
		Products:    []pmsDomain.Product{{ID: "prod-1", Name: "Açaí Bowl", Slug: "acai-bowl"}},
		Suppliers:   []suppliersDomain.Supplier{{ID: "sup-1", Name: "Açaí Distribuidora"}},
		Customers:   []customersDomain.Customer{{ID: "cust-1", FirstName: "Jane", LastName: "Doe"}},
		SalesOrders: []omsDomain.SalesOrder{{ID: "so-1", Number: "SO-10042", Status: omsDomain.SalesOrderStatusCompleted}},
	}}
	h := searchhandler.NewHandler(stub)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/search?q=acai")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "acai", stub.gotQuery)
	body := w.Body.String()
	assert.Contains(t, body, "Açaí Bowl")
	assert.Contains(t, body, "Açaí Distribuidora")
	assert.Contains(t, body, "Jane")
	assert.Contains(t, body, "SO-10042")
}

func TestGlobalSearchHandler_propagatesError(t *testing.T) {
	h := searchhandler.NewHandler(&stubGlobalSearch{err: assert.AnError})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/search?q=acai")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGlobalSearchHandler_missingQuery_rejectedByValidator(t *testing.T) {
	h := searchhandler.NewHandler(&stubGlobalSearch{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/search")

	// No OpenAPI request-validator middleware is wired in this bare test
	// router (that's cmd/api/main.go's job, see searchGroup.Use(...)) --
	// the generated wrapper still requires the "q" param to bind, so this
	// exercises the handler's own behavior on a genuinely missing param.
	require.NotEqual(t, http.StatusOK, w.Code)
}
