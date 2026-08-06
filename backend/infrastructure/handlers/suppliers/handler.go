package suppliers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gensuppliers "github.com/adirkuhn/commerceos/api/suppliers"
	appsuppliers "github.com/adirkuhn/commerceos/application/suppliers"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type listSuppliersUC interface {
	Execute(ctx context.Context) ([]suppliersDomain.Supplier, error)
}
type createSupplierUC interface {
	Execute(ctx context.Context, in appsuppliers.CreateSupplierInput) (*suppliersDomain.Supplier, error)
}
type updateSupplierUC interface {
	Execute(ctx context.Context, in appsuppliers.UpdateSupplierInput) (*suppliersDomain.Supplier, error)
}
type deleteSupplierUC interface {
	Execute(ctx context.Context, id string) error
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listSuppliers  listSuppliersUC
	createSupplier createSupplierUC
	updateSupplier updateSupplierUC
	deleteSupplier deleteSupplierUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listSuppliers listSuppliersUC,
	createSupplier createSupplierUC,
	updateSupplier updateSupplierUC,
	deleteSupplier deleteSupplierUC,
) *Handler {
	return &Handler{
		listSuppliers:  listSuppliers,
		createSupplier: createSupplier,
		updateSupplier: updateSupplier,
		deleteSupplier: deleteSupplier,
	}
}

var _ gensuppliers.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates the repo + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
func Register(rg gin.IRouter, db *gorm.DB) {
	supplierRepo := repositories.NewSupplierRepo(db)

	h := NewHandler(
		appsuppliers.NewListSuppliersUseCase(db, supplierRepo),
		appsuppliers.NewCreateSupplierUseCase(db, supplierRepo),
		appsuppliers.NewUpdateSupplierUseCase(db, supplierRepo),
		appsuppliers.NewDeleteSupplierUseCase(db, supplierRepo),
	)
	gensuppliers.RegisterHandlers(rg, h)
}

// ── Route handlers ───────────────────────────────────────────────────────────

func (h *Handler) ListSuppliers(c *gin.Context) {
	suppliersList, err := h.listSuppliers.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]gensuppliers.Supplier, len(suppliersList))
	for i, s := range suppliersList {
		resp[i] = toGenSupplier(s)
	}
	c.JSON(http.StatusOK, gensuppliers.SupplierListResponse{Data: &resp})
}

func (h *Handler) CreateSupplier(c *gin.Context) {
	var req gensuppliers.CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	leadTimeDays := ptrToIntPtr(req.LeadTimeDays)
	supplier, err := h.createSupplier.Execute(c.Request.Context(), appsuppliers.CreateSupplierInput{
		Name: req.Name, Email: strPtrOr(req.Email), Phone: strPtrOr(req.Phone), Country: strPtrOr(req.Country),
		LeadTimeDays: leadTimeDays, Status: suppliersDomain.SupplierStatus(req.Status), Notes: strPtrOr(req.Notes),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSupplier(*supplier)
	c.JSON(http.StatusCreated, gensuppliers.SupplierResponse{Data: &resp})
}

func (h *Handler) UpdateSupplier(c *gin.Context, id gensuppliers.Id) {
	var req gensuppliers.UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	leadTimeDays := ptrToIntPtr(req.LeadTimeDays)
	supplier, err := h.updateSupplier.Execute(c.Request.Context(), appsuppliers.UpdateSupplierInput{
		ID: id, Name: req.Name, Email: strPtrOr(req.Email), Phone: strPtrOr(req.Phone), Country: strPtrOr(req.Country),
		LeadTimeDays: leadTimeDays, Status: suppliersDomain.SupplierStatus(req.Status), Notes: strPtrOr(req.Notes),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSupplier(*supplier)
	c.JSON(http.StatusOK, gensuppliers.SupplierResponse{Data: &resp})
}

func (h *Handler) DeleteSupplier(c *gin.Context, id gensuppliers.Id) {
	err := h.deleteSupplier.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gensuppliers.EmptyResponse{})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToIntPtr(i *int32) *int {
	if i == nil {
		return nil
	}
	v := int(*i)
	return &v
}

func toGenSupplier(s suppliersDomain.Supplier) gensuppliers.Supplier {
	var leadTimeDays *int32
	if s.LeadTimeDays != nil {
		v := int32(*s.LeadTimeDays)
		leadTimeDays = &v
	}
	return gensuppliers.Supplier{
		Id: s.ID, TenantId: s.TenantID, Name: s.Name, Email: s.Email, Phone: s.Phone, Country: s.Country,
		LeadTimeDays: leadTimeDays, Status: gensuppliers.SupplierStatus(s.Status), Notes: s.Notes,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, suppliersDomain.ErrSupplierNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("suppliers: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
