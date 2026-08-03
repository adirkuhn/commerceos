package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateMediaInput struct {
	ID      string
	AltText string
}

type UpdateMediaUseCase struct {
	db        *gorm.DB
	mediaRepo pmsDomain.MediaRepository
}

func NewUpdateMediaUseCase(db *gorm.DB, mediaRepo pmsDomain.MediaRepository) *UpdateMediaUseCase {
	return &UpdateMediaUseCase{db: db, mediaRepo: mediaRepo}
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateMedia: %w", err)
	}
	return &media, nil
}
