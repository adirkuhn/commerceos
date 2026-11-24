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
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
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
type getCustomerUC interface {
	Execute(ctx context.Context, id string) (*customersDomain.Customer, error)
}
type deleteCustomerUC interface {
	Execute(ctx context.Context, id string) error
}
type listCustomerWhatsAppMessagesUC interface {
	Execute(ctx context.Context, customerID string) ([]whatsappDomain.Message, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listCustomers                listCustomersUC
	createCustomer               createCustomerUC
	updateCustomer               updateCustomerUC
	deleteCustomer               deleteCustomerUC
	listCustomerWhatsAppMessages listCustomerWhatsAppMessagesUC
	getCustomer                  getCustomerUC
	contactPersons               ContactPersonUseCases
}

func NewHandler(
	listCustomers listCustomersUC, createCustomer createCustomerUC, updateCustomer updateCustomerUC, deleteCustomer deleteCustomerUC,
	listCustomerWhatsAppMessages listCustomerWhatsAppMessagesUC,
	getCustomer getCustomerUC,
) *Handler {
	return &Handler{
		listCustomers: listCustomers, createCustomer: createCustomer, updateCustomer: updateCustomer, deleteCustomer: deleteCustomer,
		listCustomerWhatsAppMessages: listCustomerWhatsAppMessages, getCustomer: getCustomer,
	}
}

var _ gencustomers.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates the repo + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
func Register(rg gin.IRouter, db *gorm.DB) {
	customerRepo := repositories.NewCustomerRepo(db)
	personRepo := repositories.NewCustomerContactPersonRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)

	h := NewHandler(
		appcustomers.NewListCustomersUseCase(db, customerRepo),
		appcustomers.NewCreateCustomerUseCase(db, customerRepo, outboxRepo).WithContactPersons(personRepo),
		appcustomers.NewUpdateCustomerUseCase(db, customerRepo, outboxRepo),
		appcustomers.NewDeleteCustomerUseCase(db, customerRepo, outboxRepo),
		appcustomers.NewListCustomerWhatsAppMessagesUseCase(db, repositories.NewWhatsAppMessageRepo(db)),
		appcustomers.NewGetCustomerUseCase(db, customerRepo),
	).WithContactPersons(ContactPersonUseCases{
		List:   appcustomers.NewListContactPersonsUseCase(db, customerRepo, personRepo).Execute,
		Add:    appcustomers.NewAddContactPersonUseCase(db, customerRepo, personRepo, outboxRepo).Execute,
		Update: appcustomers.NewUpdateContactPersonUseCase(db, personRepo, outboxRepo).Execute,
		Remove: appcustomers.NewRemoveContactPersonUseCase(db, personRepo, outboxRepo).Execute,
	})
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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	customerType, ok := customerTypeFrom(req.CustomerType)
	if !ok {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", customersDomain.ErrInvalidCustomerType)
		return
	}
	cust, err := h.createCustomer.Execute(c.Request.Context(), appcustomers.CreateCustomerInput{
		WooCustomerID: req.WooCustomerId, CompanyName: strPtrOr(req.CompanyName), ClientNumber: strPtrOr(req.ClientNumber),
		KVKNumber: strPtrOr(req.KvkNumber), VATNumber: strPtrOr(req.VatNumber), Type: valueOr(customerType),
		ContactPersons: toContactPersonInputs(req.ContactPersons), Email: strPtrOr(req.Email), FirstName: strPtrOr(req.FirstName), LastName: strPtrOr(req.LastName),
		Username: strPtrOr(req.Username), Role: strPtrOr(req.Role), Billing: toDomainAddress(req.Billing), Shipping: toDomainAddress(req.Shipping),
		AvatarURL: strPtrOr(req.AvatarUrl), CustomerNote: strPtrOr(req.CustomerNote),
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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	customerType, ok := customerTypeFrom(req.CustomerType)
	if !ok {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", customersDomain.ErrInvalidCustomerType)
		return
	}
	cust, err := h.updateCustomer.Execute(c.Request.Context(), appcustomers.UpdateCustomerInput{
		ID: id, WooCustomerID: req.WooCustomerId, Type: customerType, CompanyName: req.CompanyName, ClientNumber: req.ClientNumber,
		KVKNumber: req.KvkNumber, VATNumber: req.VatNumber, Email: strPtrOr(req.Email), FirstName: strPtrOr(req.FirstName), LastName: strPtrOr(req.LastName),
		Username: strPtrOr(req.Username), Role: strPtrOr(req.Role), Billing: toDomainAddress(req.Billing), Shipping: toDomainAddress(req.Shipping),
		AvatarURL: strPtrOr(req.AvatarUrl), CustomerNote: strPtrOr(req.CustomerNote),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCustomer(*cust)
	c.JSON(http.StatusOK, gencustomers.CustomerResponse{Data: &resp})
}

func (h *Handler) GetCustomer(c *gin.Context, id gencustomers.Id) {
	cust, err := h.getCustomer.Execute(c.Request.Context(), id)
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

func (h *Handler) ListCustomerWhatsAppMessages(c *gin.Context, id gencustomers.Id) {
	list, err := h.listCustomerWhatsAppMessages.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]gencustomers.WhatsAppMessage, len(list))
	for i, m := range list {
		resp[i] = gencustomers.WhatsAppMessage{
			Id: m.ID, PhoneE164: m.PhoneE164, Direction: gencustomers.WhatsAppMessageDirection(m.Direction),
			Body: m.Body, CreatedAt: m.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, gencustomers.WhatsAppMessageListResponse{Data: &resp})
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
		CustomerType: gencustomers.CustomerType(cust.TypeOrPrivate()), DisplayName: cust.DisplayName(),
		CompanyName: cust.CompanyName, ClientNumber: cust.ClientNumber, KvkNumber: cust.KVKNumber, VatNumber: cust.VATNumber,
		Email: cust.Email, FirstName: cust.FirstName, LastName: cust.LastName,
		Username: &cust.Username, Role: &cust.Role, Billing: &billing, Shipping: &shipping,
		IsPayingCustomer: cust.IsPayingCustomer, AvatarUrl: &cust.AvatarURL, CustomerNote: &cust.CustomerNote,
		DateCreated: cust.DateCreated, DateModified: cust.DateModified,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, customersDomain.ErrCustomerNotFound), errors.Is(err, customersDomain.ErrContactPersonNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, customersDomain.ErrCustomerUnidentified), errors.Is(err, customersDomain.ErrCompanyNameRequired),
		errors.Is(err, customersDomain.ErrInvalidCustomerType), errors.Is(err, customersDomain.ErrContactPersonNameRequired):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, customersDomain.ErrDuplicateWooCustomerID), errors.Is(err, customersDomain.ErrDuplicateClientNumber):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	default:
		log.Printf("customers: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}

// customerTypeFrom reads an optional customer_type; false for an unknown one.
func customerTypeFrom(t *gencustomers.CustomerType) (*customersDomain.Type, bool) {
	if t == nil {
		return nil, true
	}
	v := customersDomain.Type(*t)
	return &v, v.Valid()
}

func valueOr[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
