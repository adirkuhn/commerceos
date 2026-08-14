package tax

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gentax "github.com/adirkuhn/commerceos/api/tax"
	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type createTaxClassUC interface {
	Execute(ctx context.Context, in apptax.CreateTaxClassInput) (*taxDomain.TaxClass, error)
}
type updateTaxClassUC interface {
	Execute(ctx context.Context, in apptax.UpdateTaxClassInput) (*taxDomain.TaxClass, error)
}
type deleteTaxClassUC interface {
	Execute(ctx context.Context, id string) error
}
type listTaxClassesUC interface {
	Execute(ctx context.Context) ([]taxDomain.TaxClass, error)
}
type upsertTaxRateUC interface {
	Execute(ctx context.Context, in apptax.UpsertTaxRateInput) (*taxDomain.TaxRate, error)
}
type deleteTaxRateUC interface {
	Execute(ctx context.Context, id string) error
}
type listTaxRatesUC interface {
	Execute(ctx context.Context) ([]taxDomain.TaxRate, error)
}
type upsertExternalTaxClassLinkUC interface {
	Execute(ctx context.Context, in apptax.UpsertExternalTaxClassLinkInput) (*taxDomain.ExternalTaxClassLink, error)
}
type deleteExternalTaxClassLinkUC interface {
	Execute(ctx context.Context, id string) error
}
type listExternalTaxClassLinksUC interface {
	Execute(ctx context.Context) ([]taxDomain.ExternalTaxClassLink, error)
}

type Handler struct {
	createTaxClass             createTaxClassUC
	updateTaxClass             updateTaxClassUC
	deleteTaxClass             deleteTaxClassUC
	listTaxClasses             listTaxClassesUC
	upsertTaxRate              upsertTaxRateUC
	deleteTaxRate              deleteTaxRateUC
	listTaxRates               listTaxRatesUC
	upsertExternalTaxClassLink upsertExternalTaxClassLinkUC
	deleteExternalTaxClassLink deleteExternalTaxClassLinkUC
	listExternalTaxClassLinks  listExternalTaxClassLinksUC
}

func NewHandler(
	createTaxClass createTaxClassUC,
	updateTaxClass updateTaxClassUC,
	deleteTaxClass deleteTaxClassUC,
	listTaxClasses listTaxClassesUC,
	upsertTaxRate upsertTaxRateUC,
	deleteTaxRate deleteTaxRateUC,
	listTaxRates listTaxRatesUC,
	upsertExternalTaxClassLink upsertExternalTaxClassLinkUC,
	deleteExternalTaxClassLink deleteExternalTaxClassLinkUC,
	listExternalTaxClassLinks listExternalTaxClassLinksUC,
) *Handler {
	return &Handler{
		createTaxClass: createTaxClass, updateTaxClass: updateTaxClass, deleteTaxClass: deleteTaxClass,
		listTaxClasses: listTaxClasses, upsertTaxRate: upsertTaxRate, deleteTaxRate: deleteTaxRate,
		listTaxRates: listTaxRates, upsertExternalTaxClassLink: upsertExternalTaxClassLink,
		deleteExternalTaxClassLink: deleteExternalTaxClassLink, listExternalTaxClassLinks: listExternalTaxClassLinks,
	}
}

var _ gentax.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	classRepo := repositories.NewTaxClassRepo(db)
	rateRepo := repositories.NewTaxRateRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	linkRepo := repositories.NewExternalTaxClassLinkRepo(db)
	h := NewHandler(
		apptax.NewCreateTaxClassUseCase(db, classRepo),
		apptax.NewUpdateTaxClassUseCase(db, classRepo),
		apptax.NewDeleteTaxClassUseCase(db, classRepo, skuRepo),
		apptax.NewListTaxClassesUseCase(db, classRepo),
		apptax.NewUpsertTaxRateUseCase(db, rateRepo, classRepo),
		apptax.NewDeleteTaxRateUseCase(db, rateRepo),
		apptax.NewListTaxRatesUseCase(db, rateRepo),
		apptax.NewUpsertExternalTaxClassLinkUseCase(db, linkRepo, classRepo),
		apptax.NewDeleteExternalTaxClassLinkUseCase(db, linkRepo),
		apptax.NewListExternalTaxClassLinksUseCase(db, linkRepo),
	)
	gentax.RegisterHandlers(rg, h)
}

func (h *Handler) ListTaxClasses(c *gin.Context) {
	classes, err := h.listTaxClasses.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTaxClasses(classes))
}

func (h *Handler) CreateTaxClass(c *gin.Context) {
	var req gentax.CreateTaxClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	class, err := h.createTaxClass.Execute(c.Request.Context(), apptax.CreateTaxClassInput{
		Name: req.Name, IsDefault: boolOr(req.IsDefault),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTaxClass(*class))
}

func (h *Handler) UpdateTaxClass(c *gin.Context, id gentax.Id) {
	var req gentax.UpdateTaxClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	class, err := h.updateTaxClass.Execute(c.Request.Context(), apptax.UpdateTaxClassInput{
		ID: id, Name: req.Name, IsDefault: boolOr(req.IsDefault),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTaxClass(*class))
}

func (h *Handler) DeleteTaxClass(c *gin.Context, id gentax.Id) {
	if err := mapError(c, h.deleteTaxClass.Execute(c.Request.Context(), id)); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListTaxRates(c *gin.Context) {
	rates, err := h.listTaxRates.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTaxRates(rates))
}

func (h *Handler) UpsertTaxRate(c *gin.Context) {
	var req gentax.UpsertTaxRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	rate, err := h.upsertTaxRate.Execute(c.Request.Context(), apptax.UpsertTaxRateInput{
		TaxClassID: req.TaxClassId, Country: req.Country, RateBasisPoints: req.RateBasisPoints,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTaxRate(*rate))
}

func (h *Handler) DeleteTaxRate(c *gin.Context, id gentax.Id) {
	if err := mapError(c, h.deleteTaxRate.Execute(c.Request.Context(), id)); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListExternalTaxClassLinks(c *gin.Context) {
	links, err := h.listExternalTaxClassLinks.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenExternalTaxClassLinks(links))
}

func (h *Handler) UpsertExternalTaxClassLink(c *gin.Context) {
	var req gentax.UpsertExternalTaxClassLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	link, err := h.upsertExternalTaxClassLink.Execute(c.Request.Context(), apptax.UpsertExternalTaxClassLinkInput{
		Provider: req.Provider, ExternalSlug: req.ExternalSlug, TaxClassID: req.TaxClassId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenExternalTaxClassLink(*link))
}

func (h *Handler) DeleteExternalTaxClassLink(c *gin.Context, id gentax.Id) {
	if err := mapError(c, h.deleteExternalTaxClassLink.Execute(c.Request.Context(), id)); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func boolOr(b *bool) bool {
	return b != nil && *b
}

func toGenTaxClass(c taxDomain.TaxClass) gentax.TaxClass {
	return gentax.TaxClass{
		Id: c.ID, Name: c.Name, IsDefault: c.IsDefault, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toGenTaxClasses(classes []taxDomain.TaxClass) []gentax.TaxClass {
	out := make([]gentax.TaxClass, len(classes))
	for i, c := range classes {
		out[i] = toGenTaxClass(c)
	}
	return out
}

func toGenTaxRate(r taxDomain.TaxRate) gentax.TaxRate {
	return gentax.TaxRate{
		Id: r.ID, TaxClassId: r.TaxClassID, Country: r.Country, RateBasisPoints: r.RateBasisPoints,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toGenTaxRates(rates []taxDomain.TaxRate) []gentax.TaxRate {
	out := make([]gentax.TaxRate, len(rates))
	for i, r := range rates {
		out[i] = toGenTaxRate(r)
	}
	return out
}

func toGenExternalTaxClassLink(l taxDomain.ExternalTaxClassLink) gentax.ExternalTaxClassLink {
	return gentax.ExternalTaxClassLink{
		Id: l.ID, Provider: l.Provider, ExternalSlug: l.ExternalSlug, TaxClassId: l.TaxClassID,
		CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
}

func toGenExternalTaxClassLinks(links []taxDomain.ExternalTaxClassLink) []gentax.ExternalTaxClassLink {
	out := make([]gentax.ExternalTaxClassLink, len(links))
	for i, l := range links {
		out[i] = toGenExternalTaxClassLink(l)
	}
	return out
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, taxDomain.ErrTaxClassNotFound),
		errors.Is(err, taxDomain.ErrTaxRateNotFound),
		errors.Is(err, taxDomain.ErrExternalTaxClassLinkNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, taxDomain.ErrTaxClassNameRequired),
		errors.Is(err, taxDomain.ErrCannotDeleteDefaultTaxClass),
		errors.Is(err, taxDomain.ErrTaxClassInUse),
		errors.Is(err, taxDomain.ErrInvalidCountryCode),
		errors.Is(err, taxDomain.ErrInvalidTaxRateValue),
		errors.Is(err, taxDomain.ErrProviderRequired):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("tax: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
