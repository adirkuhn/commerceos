package customers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listCustomersUC interface {
	Execute(ctx context.Context) ([]customersDomain.Customer, error)
}
type createCustomerUC interface {
	Execute(ctx context.Context, in appcustomers.CreateCustomerInput) (*customersDomain.Customer, error)
}
type updateCustomerUC interface {
	Execute(ctx context.Context, in appcustomers.UpdateCustomerInput) (*customersDomain.Customer, error)
}
type deleteCustomerUC interface {
	Execute(ctx context.Context, id string) error
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listCustomers  listCustomersUC
	createCustomer createCustomerUC
	updateCustomer updateCustomerUC
	deleteCustomer deleteCustomerUC
}

func NewHandler(listCustomers listCustomersUC, createCustomer createCustomerUC, updateCustomer updateCustomerUC, deleteCustomer deleteCustomerUC) *Handler {
	return &Handler{listCustomers: listCustomers, createCustomer: createCustomer, updateCustomer: updateCustomer, deleteCustomer: deleteCustomer}
}

var _ gencustomers.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates the repo + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
func Register(rg gin.IRouter, db *gorm.DB) {
	customerRepo := repositories.NewCustomerRepo(db)

	h := NewHandler(
		appcustomers.NewListCustomersUseCase(db, customerRepo),
		appcustomers.NewCreateCustomerUseCase(db, customerRepo),
		appcustomers.NewUpdateCustomerUseCase(db, customerRepo),
		appcustomers.NewDeleteCustomerUseCase(db, customerRepo),
	)
	gencustomers.RegisterHandlers(rg, h)
}

// ── Route handlers ───────────────────────────────────────────────────────────

func (h *Handler) ListCustomers(c *gin.Context) {
	list, err := h.listCustomers.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]gencustomers.Customer, len(list))
	for i, cust := range list {
		resp[i] = toGenCustomer(cust)
	}
	c.JSON(http.StatusOK, gencustomers.CustomerListResponse{Data: &resp})
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	var req gencustomers.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Email == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email is required")
		return
	}
	cust, err := h.createCustomer.Execute(c.Request.Context(), appcustomers.CreateCustomerInput{
		WooCustomerID: req.WooCustomerId, Email: req.Email, FirstName: strPtrOr(req.FirstName), LastName: strPtrOr(req.LastName),
		Username: strPtrOr(req.Username), Role: strPtrOr(req.Role), Billing: toDomainAddress(req.Billing), Shipping: toDomainAddress(req.Shipping),
		IsPayingCustomer: boolPtrOr(req.IsPayingCustomer), AvatarURL: strPtrOr(req.AvatarUrl),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCustomer(*cust)
	c.JSON(http.StatusCreated, gencustomers.CustomerResponse{Data: &resp})
}

func (h *Handler) UpdateCustomer(c *gin.Context, id gencustomers.Id) {
	var req gencustomers.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Email == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email is required")
		return
	}
	cust, err := h.updateCustomer.Execute(c.Request.Context(), appcustomers.UpdateCustomerInput{
		ID: id, WooCustomerID: req.WooCustomerId, Email: req.Email, FirstName: strPtrOr(req.FirstName), LastName: strPtrOr(req.LastName),
		Username: strPtrOr(req.Username), Role: strPtrOr(req.Role), Billing: toDomainAddress(req.Billing), Shipping: toDomainAddress(req.Shipping),
		IsPayingCustomer: boolPtrOr(req.IsPayingCustomer), AvatarURL: strPtrOr(req.AvatarUrl),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCustomer(*cust)
	c.JSON(http.StatusOK, gencustomers.CustomerResponse{Data: &resp})
}

func (h *Handler) DeleteCustomer(c *gin.Context, id gencustomers.Id) {
	err := h.deleteCustomer.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gencustomers.EmptyResponse{})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolPtrOr(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func toDomainAddress(a *gencustomers.Address) customersDomain.Address {
	if a == nil {
		return customersDomain.Address{}
	}
	return customersDomain.Address{
		FirstName: strPtrOr(a.FirstName), LastName: strPtrOr(a.LastName), Company: strPtrOr(a.Company),
		Address1: strPtrOr(a.Address1), Address2: strPtrOr(a.Address2), City: strPtrOr(a.City),
		State: strPtrOr(a.State), Postcode: strPtrOr(a.Postcode), Country: strPtrOr(a.Country),
		Email: strPtrOr(a.Email), Phone: strPtrOr(a.Phone),
	}
}

func toGenAddress(a customersDomain.Address) gencustomers.Address {
	return gencustomers.Address{
		FirstName: &a.FirstName, LastName: &a.LastName, Company: &a.Company,
		Address1: &a.Address1, Address2: &a.Address2, City: &a.City,
		State: &a.State, Postcode: &a.Postcode, Country: &a.Country,
		Email: &a.Email, Phone: &a.Phone,
	}
}

func toGenCustomer(cust customersDomain.Customer) gencustomers.Customer {
	billing := toGenAddress(cust.Billing)
	shipping := toGenAddress(cust.Shipping)
	return gencustomers.Customer{
		Id: cust.ID, TenantId: cust.TenantID, WooCustomerId: cust.WooCustomerID,
		Email: cust.Email, FirstName: cust.FirstName, LastName: cust.LastName,
		Username: &cust.Username, Role: &cust.Role, Billing: &billing, Shipping: &shipping,
		IsPayingCustomer: cust.IsPayingCustomer, AvatarUrl: &cust.AvatarURL,
		DateCreated: cust.DateCreated, DateModified: cust.DateModified,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, customersDomain.ErrCustomerNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, customersDomain.ErrDuplicateWooCustomerID):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	default:
		log.Printf("customers: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
