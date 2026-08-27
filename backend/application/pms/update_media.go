package pms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateMediaInput struct {
	ID      string
	AltText string
}

type UpdateMediaUseCase struct {
	db         *gorm.DB
	mediaRepo  pmsDomain.MediaRepository
	outboxRepo outbox.Repository
}

func NewUpdateMediaUseCase(db *gorm.DB, mediaRepo pmsDomain.MediaRepository, outboxRepo outbox.Repository) *UpdateMediaUseCase {
	return &UpdateMediaUseCase{db: db, mediaRepo: mediaRepo, outboxRepo: outboxRepo}
}

func (uc *UpdateMediaUseCase) Execute(ctx context.Context, in UpdateMediaInput) (*pmsDomain.Media, error) {
	var media pmsDomain.Media
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.mediaRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.AltText = in.AltText
		if err := uc.mediaRepo.Update(ctx, existing); err != nil {
			return err
		}
		media = *existing

		// Only product-scoped media (ProductID set) ever reaches Woo --
		// wooImagesFromMedia is built from mediaRepo.ListByProduct, which
		// filters strictly on product_id, so a variant-only media row
		// (ProductID nil, used solely as a variant's featured-image pick)
		// never appears in a push regardless of its alt text.
		if existing.ProductID == nil {
			return nil
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, *existing.ProductID)
	})
	if err != nil {
		return nil, fmt.Errorf("updateMedia: %w", err)
	}
	return &media, nil
}
