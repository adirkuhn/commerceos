package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type bolProductContentModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	ProductID    string `gorm:"not null;type:uuid"`
	ConnectionID string `gorm:"not null;type:uuid"`

	ManufacturerContact   *string
	ManufacturerAddress   *string
	ManufacturerName      *string
	ShelfLifeClosed       *string
	ShelfLifeAfterOpening *string
	Ingredients           *string
	ContentAmount         *string
	ContentUnit           *string
	CountryOfOrigin       *string
	Substance             *string
	InstructionLanguage   *string
	PackagingLanguage     *string

	ItemsPerPackage           *int32
	PackageInsertURL          *string
	IsGift                    *bool
	DermatologicallyTested    *bool
	HazardStatements          *string
	SkinType                  *string
	Hypoallergenic            *bool
	ContainsZincOxide         *bool
	ContentPerDispenserAmount *string
	ContentPerDispenserUnit   *string
	Brand                     *string
	FragranceFree             *bool
	PHNeutral                 *bool `gorm:"column:ph_neutral"`
	CoolingEffect             *bool
	PackageWidthMM            *int32
	PackageHeightMM           *int32
	PackageLengthMM           *int32
	PackageWeightG            *int32
	PrecautionStatements      *string
	Precautions               *string
	Warnings                  *string
	VideoURL                  *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (bolProductContentModel) TableName() string { return "bol_product_content" }

type BolProductContentRepo struct{ db *gorm.DB }

func NewBolProductContentRepo(db *gorm.DB) *BolProductContentRepo {
	return &BolProductContentRepo{db: db}
}

func (r *BolProductContentRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("bolProductContent: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BolProductContentRepo) FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*channelsDomain.BolProductContent, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m bolProductContentModel
	err = tx.Where("product_id = ? AND connection_id = ?", productID, connectionID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bolProductContentRepo.FindByProductAndConnection: %w", err)
	}
	return toBolProductContentDomain(m), nil
}

// Upsert inserts or updates by the (product_id, connection_id) unique
// constraint (migration 107).
func (r *BolProductContentRepo) Upsert(ctx context.Context, c *channelsDomain.BolProductContent) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := bolProductContentModel{
		TenantID: c.TenantID, ProductID: c.ProductID, ConnectionID: c.ConnectionID,
		ManufacturerContact: c.ManufacturerContact, ManufacturerAddress: c.ManufacturerAddress, ManufacturerName: c.ManufacturerName,
		ShelfLifeClosed: c.ShelfLifeClosed, ShelfLifeAfterOpening: c.ShelfLifeAfterOpening, Ingredients: c.Ingredients,
		ContentAmount: c.ContentAmount, ContentUnit: c.ContentUnit, CountryOfOrigin: c.CountryOfOrigin,
		Substance: c.Substance, InstructionLanguage: c.InstructionLanguage, PackagingLanguage: c.PackagingLanguage,
		ItemsPerPackage: c.ItemsPerPackage, PackageInsertURL: c.PackageInsertURL, IsGift: c.IsGift,
		DermatologicallyTested: c.DermatologicallyTested, HazardStatements: c.HazardStatements, SkinType: c.SkinType,
		Hypoallergenic: c.Hypoallergenic, ContainsZincOxide: c.ContainsZincOxide,
		ContentPerDispenserAmount: c.ContentPerDispenserAmount, ContentPerDispenserUnit: c.ContentPerDispenserUnit,
		Brand: c.Brand, FragranceFree: c.FragranceFree, PHNeutral: c.PHNeutral, CoolingEffect: c.CoolingEffect,
		PackageWidthMM: c.PackageWidthMM, PackageHeightMM: c.PackageHeightMM, PackageLengthMM: c.PackageLengthMM, PackageWeightG: c.PackageWeightG,
		PrecautionStatements: c.PrecautionStatements, Precautions: c.Precautions, Warnings: c.Warnings, VideoURL: c.VideoURL,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "product_id"}, {Name: "connection_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"manufacturer_contact", "manufacturer_address", "manufacturer_name",
			"shelf_life_closed", "shelf_life_after_opening", "ingredients",
			"content_amount", "content_unit", "country_of_origin",
			"substance", "instruction_language", "packaging_language",
			"items_per_package", "package_insert_url", "is_gift",
			"dermatologically_tested", "hazard_statements", "skin_type",
			"hypoallergenic", "contains_zinc_oxide",
			"content_per_dispenser_amount", "content_per_dispenser_unit",
			"brand", "fragrance_free", "ph_neutral", "cooling_effect",
			"package_width_mm", "package_height_mm", "package_length_mm", "package_weight_g",
			"precaution_statements", "precautions", "warnings", "video_url",
			"updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("bolProductContentRepo.Upsert: %w", err)
	}
	c.ID = m.ID
	return nil
}

func toBolProductContentDomain(m bolProductContentModel) *channelsDomain.BolProductContent {
	return &channelsDomain.BolProductContent{
		ID: m.ID, TenantID: m.TenantID, ProductID: m.ProductID, ConnectionID: m.ConnectionID,
		ManufacturerContact: m.ManufacturerContact, ManufacturerAddress: m.ManufacturerAddress, ManufacturerName: m.ManufacturerName,
		ShelfLifeClosed: m.ShelfLifeClosed, ShelfLifeAfterOpening: m.ShelfLifeAfterOpening, Ingredients: m.Ingredients,
		ContentAmount: m.ContentAmount, ContentUnit: m.ContentUnit, CountryOfOrigin: m.CountryOfOrigin,
		Substance: m.Substance, InstructionLanguage: m.InstructionLanguage, PackagingLanguage: m.PackagingLanguage,
		ItemsPerPackage: m.ItemsPerPackage, PackageInsertURL: m.PackageInsertURL, IsGift: m.IsGift,
		DermatologicallyTested: m.DermatologicallyTested, HazardStatements: m.HazardStatements, SkinType: m.SkinType,
		Hypoallergenic: m.Hypoallergenic, ContainsZincOxide: m.ContainsZincOxide,
		ContentPerDispenserAmount: m.ContentPerDispenserAmount, ContentPerDispenserUnit: m.ContentPerDispenserUnit,
		Brand: m.Brand, FragranceFree: m.FragranceFree, PHNeutral: m.PHNeutral, CoolingEffect: m.CoolingEffect,
		PackageWidthMM: m.PackageWidthMM, PackageHeightMM: m.PackageHeightMM, PackageLengthMM: m.PackageLengthMM, PackageWeightG: m.PackageWeightG,
		PrecautionStatements: m.PrecautionStatements, Precautions: m.Precautions, Warnings: m.Warnings, VideoURL: m.VideoURL,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
