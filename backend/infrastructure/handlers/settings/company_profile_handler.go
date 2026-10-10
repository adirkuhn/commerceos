package settings

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	gensettings "github.com/adirkuhn/commerceos/api/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type getCompanyProfileUC interface {
	Execute(ctx context.Context) (*settingsDomain.CompanyProfile, error)
}
type updateCompanyProfileUC interface {
	Execute(ctx context.Context, in settingsDomain.CompanyProfileInput) (*settingsDomain.CompanyProfile, error)
}

// CompanyProfileHandler serves /settings/company-profile. Embedded in
// Handler so the generated settings ServerInterface stays one type, while
// company-profile mapping lives here.
type CompanyProfileHandler struct {
	get    getCompanyProfileUC
	update updateCompanyProfileUC
}

func NewCompanyProfileHandler(get getCompanyProfileUC, update updateCompanyProfileUC) *CompanyProfileHandler {
	return &CompanyProfileHandler{get: get, update: update}
}

func (h *CompanyProfileHandler) GetCompanyProfile(c *gin.Context) {
	p, err := h.get.Execute(c.Request.Context())
	if err != nil {
		failCompanyProfile(c, err)
		return
	}
	resp := toGenCompanyProfile(*p)
	c.JSON(http.StatusOK, gensettings.CompanyProfileResponse{Data: &resp})
}

func (h *CompanyProfileHandler) UpdateCompanyProfile(c *gin.Context) {
	var req gensettings.UpdateCompanyProfileJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	p, err := h.update.Execute(c.Request.Context(), settingsDomain.CompanyProfileInput{
		LegalName: req.LegalName, TradingName: req.TradingName, VATNumber: req.VatNumber,
		ChamberOfCommerce: req.ChamberOfCommerce, Email: req.Email, Phone: req.Phone,
		Address: settingsDomain.CompanyAddress{
			Street: req.Address.Street, HouseNumber: req.Address.HouseNumber, PostalCode: req.Address.PostalCode,
			City: req.Address.City, Country: req.Address.Country,
		},
		IBAN: req.Iban,
	})
	if err != nil {
		failCompanyProfile(c, err)
		return
	}
	resp := toGenCompanyProfile(*p)
	c.JSON(http.StatusOK, gensettings.CompanyProfileResponse{Data: &resp})
}

// companyProfileValidationErrors answer 422 with their own code
// (domain/errs), e.g. COMPANY_INVALID_IBAN.
var companyProfileValidationErrors = []error{
	settingsDomain.ErrLegalNameRequired,
	settingsDomain.ErrInvalidCompanyEmail,
	settingsDomain.ErrInvalidVATNumber,
	settingsDomain.ErrInvalidIBAN,
	settingsDomain.ErrInvalidCountryCode,
}

func failCompanyProfile(c *gin.Context, err error) {
	for _, known := range companyProfileValidationErrors {
		if errors.Is(err, known) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
	}
	log.Printf("settings: company profile: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
}

func toGenCompanyProfile(p settingsDomain.CompanyProfile) gensettings.CompanyProfile {
	return gensettings.CompanyProfile{
		LegalName: p.LegalName, TradingName: p.TradingName, VatNumber: p.VATNumber,
		ChamberOfCommerce: p.ChamberOfCommerce, Email: p.Email, Phone: p.Phone,
		Address: gensettings.CompanyAddress{
			Street: p.Address.Street, HouseNumber: p.Address.HouseNumber, PostalCode: p.Address.PostalCode,
			City: p.Address.City, Country: p.Address.Country,
		},
		Iban: p.IBAN, UpdatedAt: p.UpdatedAt,
	}
}
