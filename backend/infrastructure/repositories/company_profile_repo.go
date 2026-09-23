package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type companyProfileModel struct {
	TenantID           string `gorm:"primaryKey;type:uuid"`
	LegalName          string `gorm:"column:legal_name"`
	TradingName        string `gorm:"column:trading_name"`
	VATNumber          string `gorm:"column:vat_number"`
	ChamberOfCommerce  string `gorm:"column:chamber_of_commerce"`
	Email              string `gorm:"column:email"`
	Phone              string `gorm:"column:phone"`
	AddressStreet      string `gorm:"column:address_street"`
	AddressHouseNumber string `gorm:"column:address_house_number"`
	AddressPostalCode  string `gorm:"column:address_postal_code"`
	AddressCity        string `gorm:"column:address_city"`
	AddressCountry     string `gorm:"column:address_country"`
	IBAN               string `gorm:"column:iban"`
	UpdatedAt          time.Time
}

func (companyProfileModel) TableName() string { return "company_profiles" }

type CompanyProfileRepo struct{ db *gorm.DB }

func NewCompanyProfileRepo(db *gorm.DB) *CompanyProfileRepo { return &CompanyProfileRepo{db: db} }

func (r *CompanyProfileRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("companyProfileRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Get returns the tenant's profile, or an empty one if none is saved yet --
// RLS already scopes this to at most one row.
func (r *CompanyProfileRepo) Get(ctx context.Context) (*settingsDomain.CompanyProfile, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m companyProfileModel
	if err := tx.Limit(1).Find(&m).Error; err != nil {
		return nil, fmt.Errorf("companyProfileRepo.Get: %w", err)
	}
	return &settingsDomain.CompanyProfile{
		TenantID: m.TenantID, LegalName: m.LegalName, TradingName: m.TradingName,
		VATNumber: m.VATNumber, ChamberOfCommerce: m.ChamberOfCommerce, Email: m.Email, Phone: m.Phone,
		Address: settingsDomain.CompanyAddress{
			Street: m.AddressStreet, HouseNumber: m.AddressHouseNumber, PostalCode: m.AddressPostalCode,
			City: m.AddressCity, Country: m.AddressCountry,
		},
		IBAN: m.IBAN, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *CompanyProfileRepo) Upsert(ctx context.Context, p *settingsDomain.CompanyProfile) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := companyProfileModel{
		TenantID: p.TenantID, LegalName: p.LegalName, TradingName: p.TradingName,
		VATNumber: p.VATNumber, ChamberOfCommerce: p.ChamberOfCommerce, Email: p.Email, Phone: p.Phone,
		AddressStreet: p.Address.Street, AddressHouseNumber: p.Address.HouseNumber,
		AddressPostalCode: p.Address.PostalCode, AddressCity: p.Address.City, AddressCountry: p.Address.Country,
		IBAN: p.IBAN, UpdatedAt: p.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"legal_name", "trading_name", "vat_number", "chamber_of_commerce", "email", "phone",
			"address_street", "address_house_number", "address_postal_code", "address_city", "address_country",
			"iban", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("companyProfileRepo.Upsert: %w", err)
	}
	return nil
}
