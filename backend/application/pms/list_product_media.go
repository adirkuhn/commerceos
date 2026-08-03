package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListProductMediaUseCase struct {
	db        *gorm.DB
	mediaRepo pmsDomain.MediaRepository
}

func NewListProductMediaUseCase(db *gorm.DB, mediaRepo pmsDomain.MediaRepository) *ListProductMediaUseCase {
	return &ListProductMediaUseCase{db: db, mediaRepo: mediaRepo}
}

func (uc *ListProductMediaUseCase) Execute(ctx context.Context, productID string) ([]pmsDomain.Media, error) {
	var media []pmsDomain.Media
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		media, err = uc.mediaRepo.ListByProduct(ctx, productID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listProductMedia: %w", err)
	}
	return media, nil
}
