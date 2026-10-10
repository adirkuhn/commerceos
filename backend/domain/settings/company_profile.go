package settings

import (
	"context"
	"github.com/adirkuhn/commerceos/domain/errs"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var (
	ErrLegalNameRequired   = errs.New("COMPANY_LEGAL_NAME_REQUIRED", "company legal name is required")
	ErrInvalidCompanyEmail = errs.New("COMPANY_INVALID_EMAIL", "company email is not a valid address")
	ErrInvalidVATNumber    = errs.New("COMPANY_INVALID_VAT_NUMBER", "VAT number must be a 2-letter country prefix followed by 2-13 letters or digits")
	ErrInvalidIBAN         = errs.New("COMPANY_INVALID_IBAN", "IBAN is not valid")
	ErrInvalidCountryCode  = errs.New("COMPANY_INVALID_COUNTRY", "country must be a 2-letter ISO 3166 code")
)

// CompanyAddress is the seller's registered address, as printed on invoices.
type CompanyAddress struct {
	Street      string
	HouseNumber string
	PostalCode  string
	City        string
	Country     string // ISO 3166-1 alpha-2
}

// CompanyProfile is the tenant's own business identity -- the seller block
// on invoices and the sender on customer communication. Separate from
// TenantSettings (localization/behaviour defaults): different data, a
// different reason to change.
type CompanyProfile struct {
	TenantID          string
	LegalName         string
	TradingName       string
	VATNumber         string
	ChamberOfCommerce string
	Email             string
	Phone             string
	Address           CompanyAddress
	IBAN              string
	UpdatedAt         time.Time
}

// CompanyProfileInput is what a caller may set; NewCompanyProfile
// normalizes and validates it.
type CompanyProfileInput struct {
	LegalName         string
	TradingName       string
	VATNumber         string
	ChamberOfCommerce string
	Email             string
	Phone             string
	Address           CompanyAddress
	IBAN              string
}

type CompanyProfileRepository interface {
	// Get returns the tenant's profile, or an empty one (zero TenantID) if
	// none has been saved yet.
	Get(ctx context.Context) (*CompanyProfile, error)
	Upsert(ctx context.Context, p *CompanyProfile) error
}

var (
	vatPattern     = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{2,13}$`)
	ibanPattern    = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$`)
	countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)
	// VAT and IBAN are commonly written with spaces, dots or dashes.
	separators = strings.NewReplacer(" ", "", ".", "", "-", "")
)

func NewCompanyProfile(tenantID string, in CompanyProfileInput, now time.Time) (*CompanyProfile, error) {
	p := &CompanyProfile{
		TenantID:          tenantID,
		LegalName:         strings.TrimSpace(in.LegalName),
		TradingName:       strings.TrimSpace(in.TradingName),
		VATNumber:         strings.ToUpper(separators.Replace(in.VATNumber)),
		ChamberOfCommerce: strings.TrimSpace(in.ChamberOfCommerce),
		Email:             strings.TrimSpace(in.Email),
		Phone:             strings.TrimSpace(in.Phone),
		Address: CompanyAddress{
			Street:      strings.TrimSpace(in.Address.Street),
			HouseNumber: strings.TrimSpace(in.Address.HouseNumber),
			PostalCode:  strings.TrimSpace(in.Address.PostalCode),
			City:        strings.TrimSpace(in.Address.City),
			Country:     strings.ToUpper(strings.TrimSpace(in.Address.Country)),
		},
		IBAN:      strings.ToUpper(separators.Replace(in.IBAN)),
		UpdatedAt: now,
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *CompanyProfile) validate() error {
	if p.LegalName == "" {
		return ErrLegalNameRequired
	}
	if p.Email != "" && !isPlainEmail(p.Email) {
		return ErrInvalidCompanyEmail
	}
	if p.VATNumber != "" && !vatPattern.MatchString(p.VATNumber) {
		return ErrInvalidVATNumber
	}
	if p.IBAN != "" && !isValidIBAN(p.IBAN) {
		return ErrInvalidIBAN
	}
	if p.Address.Country != "" && !countryPattern.MatchString(p.Address.Country) {
		return ErrInvalidCountryCode
	}
	return nil
}

// isPlainEmail accepts a bare address only ("a@b.c"), not "Name <a@b.c>".
func isPlainEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && strings.Contains(s[strings.LastIndex(s, "@"):], ".")
}

// isValidIBAN checks the shape and the ISO 13616 mod-97 checksum.
func isValidIBAN(iban string) bool {
	if !ibanPattern.MatchString(iban) {
		return false
	}
	rearranged := iban[4:] + iban[:4]
	var digits strings.Builder
	for _, r := range rearranged {
		if r >= 'A' && r <= 'Z' {
			digits.WriteString(big.NewInt(int64(r - 'A' + 10)).String())
		} else {
			digits.WriteRune(r)
		}
	}
	n, ok := new(big.Int).SetString(digits.String(), 10)
	return ok && new(big.Int).Mod(n, big.NewInt(97)).Int64() == 1
}
