package pms

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateVariantInput struct {
	ID       string
	Name     string
	Position int
	// FeaturedMediaID: nil clears the variant's selected image, a set
	// value must reference a Media row belonging to the variant's product.
	FeaturedMediaID *string
}

type UpdateVariantUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
	mediaRepo   pmsDomain.MediaRepository
}

func NewUpdateVariantUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, mediaRepo pmsDomain.MediaRepository) *UpdateVariantUseCase {
	return &UpdateVariantUseCase{db: db, variantRepo: variantRepo, mediaRepo: mediaRepo}
}

func (uc *UpdateVariantUseCase) Execute(ctx context.Context, in UpdateVariantInput) (*pmsDomain.Variant, error) {
	var variant pmsDomain.Variant
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.variantRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}

		if in.FeaturedMediaID != nil {
			media, err := uc.mediaRepo.FindByID(ctx, *in.FeaturedMediaID)
			if err != nil {
				return err
			}
			if media.ProductID == nil || *media.ProductID != existing.ProductID {
				return pmsDomain.ErrFeaturedMediaProductMismatch
			}
		}

		existing.Name = in.Name
		existing.Position = in.Position
		existing.FeaturedMediaID = in.FeaturedMediaID
		existing.UpdatedAt = time.Now()
		if err := uc.variantRepo.Update(ctx, existing); err != nil {
			return err
		}
		variant = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateVariant: %w", err)
	}
	return &variant, nil
}
