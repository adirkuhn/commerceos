package channels

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetBolProductContentUseCase struct {
	db   *gorm.DB
	repo channelsDomain.BolProductContentRepository
}

func NewGetBolProductContentUseCase(db *gorm.DB, repo channelsDomain.BolProductContentRepository) *GetBolProductContentUseCase {
	return &GetBolProductContentUseCase{db: db, repo: repo}
}

// Execute returns nil, nil when no content has been filled in yet -- the
// common case for a product with no Bol product-content configured.
func (uc *GetBolProductContentUseCase) Execute(ctx context.Context, productID, connectionID string) (*channelsDomain.BolProductContent, error) {
	var result *channelsDomain.BolProductContent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.repo.FindByProductAndConnection(ctx, productID, connectionID)
		if err != nil {
			return err
		}
		result = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getBolProductContent: %w", err)
	}
	return result, nil
}

type UpsertBolProductContentInput struct {
	ProductID    string
	ConnectionID string

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
	PHNeutral                 *bool
	CoolingEffect             *bool
	PackageWidthMM            *int32
	PackageHeightMM           *int32
	PackageLengthMM           *int32
	PackageWeightG            *int32
	PrecautionStatements      *string
	Precautions               *string
	Warnings                  *string
	VideoURL                  *string
}

type UpsertBolProductContentUseCase struct {
	db   *gorm.DB
	repo channelsDomain.BolProductContentRepository
}

func NewUpsertBolProductContentUseCase(db *gorm.DB, repo channelsDomain.BolProductContentRepository) *UpsertBolProductContentUseCase {
	return &UpsertBolProductContentUseCase{db: db, repo: repo}
}

func (uc *UpsertBolProductContentUseCase) Execute(ctx context.Context, in UpsertBolProductContentInput) (*channelsDomain.BolProductContent, error) {
	content := channelsDomain.BolProductContent{
		ProductID: in.ProductID, ConnectionID: in.ConnectionID,
		ManufacturerContact: in.ManufacturerContact, ManufacturerAddress: in.ManufacturerAddress, ManufacturerName: in.ManufacturerName,
		ShelfLifeClosed: in.ShelfLifeClosed, ShelfLifeAfterOpening: in.ShelfLifeAfterOpening, Ingredients: in.Ingredients,
		ContentAmount: in.ContentAmount, ContentUnit: in.ContentUnit, CountryOfOrigin: in.CountryOfOrigin,
		Substance: in.Substance, InstructionLanguage: in.InstructionLanguage, PackagingLanguage: in.PackagingLanguage,
		ItemsPerPackage: in.ItemsPerPackage, PackageInsertURL: in.PackageInsertURL, IsGift: in.IsGift,
		DermatologicallyTested: in.DermatologicallyTested, HazardStatements: in.HazardStatements, SkinType: in.SkinType,
		Hypoallergenic: in.Hypoallergenic, ContainsZincOxide: in.ContainsZincOxide,
		ContentPerDispenserAmount: in.ContentPerDispenserAmount, ContentPerDispenserUnit: in.ContentPerDispenserUnit,
		Brand: in.Brand, FragranceFree: in.FragranceFree, PHNeutral: in.PHNeutral, CoolingEffect: in.CoolingEffect,
		PackageWidthMM: in.PackageWidthMM, PackageHeightMM: in.PackageHeightMM, PackageLengthMM: in.PackageLengthMM, PackageWeightG: in.PackageWeightG,
		PrecautionStatements: in.PrecautionStatements, Precautions: in.Precautions, Warnings: in.Warnings, VideoURL: in.VideoURL,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		content.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		if content.CreatedAt.IsZero() {
			content.CreatedAt = now
		}
		content.UpdatedAt = now
		return uc.repo.Upsert(ctx, &content)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertBolProductContent: %w", err)
	}
	return &content, nil
}
