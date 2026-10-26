package search

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gensearch "github.com/adirkuhn/commerceos/api/search"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type globalSearchUC interface {
	Execute(ctx context.Context, query string) (*appsearch.GlobalSearchResult, error)
}

type Handler struct {
	globalSearch globalSearchUC
}

func NewHandler(globalSearch globalSearchUC) *Handler {
	return &Handler{globalSearch: globalSearch}
}

var _ gensearch.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	uc := appsearch.NewGlobalSearchUseCase(
		db,
		repositories.NewProductRepo(db),
		repositories.NewSupplierRepo(db),
		repositories.NewCustomerRepo(db),
		repositories.NewSalesOrderRepo(db),
	)
	h := NewHandler(uc)
	gensearch.RegisterHandlers(rg, h)
}

func (h *Handler) GlobalSearch(c *gin.Context, params gensearch.GlobalSearchParams) {
	result, err := h.globalSearch.Execute(c.Request.Context(), params.Q)
	if err != nil {
		log.Printf("search: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	c.JSON(http.StatusOK, gensearch.GlobalSearchResponse{Data: toGenResult(*result)})
}

func toGenResult(r appsearch.GlobalSearchResult) *gensearch.GlobalSearchResult {
	products := make([]gensearch.SearchProductResult, len(r.Products))
	for i, p := range r.Products {
		products[i] = gensearch.SearchProductResult{Id: p.ID, Name: p.Name, Slug: p.Slug}
	}
	suppliers := make([]gensearch.SearchSupplierResult, len(r.Suppliers))
	for i, s := range r.Suppliers {
		email := s.Email
		suppliers[i] = gensearch.SearchSupplierResult{Id: s.ID, Name: s.Name, Email: &email}
	}
	customers := make([]gensearch.SearchCustomerResult, len(r.Customers))
	for i, c := range r.Customers {
		email := c.Email
		customers[i] = gensearch.SearchCustomerResult{Id: c.ID, FirstName: c.FirstName, LastName: c.LastName, Email: &email,
			DisplayName: c.DisplayName(), CustomerType: gensearch.SearchCustomerResultCustomerType(c.TypeOrPrivate()),
			CompanyName: c.CompanyName, ClientNumber: c.ClientNumber}
	}
	salesOrders := make([]gensearch.SearchSalesOrderResult, len(r.SalesOrders))
	for i, o := range r.SalesOrders {
		billingFirst, billingLast := o.Billing.FirstName, o.Billing.LastName
		salesOrders[i] = gensearch.SearchSalesOrderResult{
			Id: o.ID, Number: o.Number, Status: string(o.Status),
			BillingFirstName: &billingFirst, BillingLastName: &billingLast,
		}
	}
	return &gensearch.GlobalSearchResult{
		Products: products, Suppliers: suppliers, Customers: customers, SalesOrders: salesOrders,
	}
}
