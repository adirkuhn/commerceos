package pms

import (
	"context"
	"fmt"
	"io"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var allowedImageContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type UploadMediaInput struct {
	ProductID   *string
	VariantID   *string
	Filename    string
	ContentType string
	Data        io.Reader
	AltText     string
}

type UploadMediaUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	mediaRepo   pmsDomain.MediaRepository
	storage     pmsDomain.FileStorage
}

func NewUploadMediaUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	mediaRepo pmsDomain.MediaRepository,
	storage pmsDomain.FileStorage,
) *UploadMediaUseCase {
	return &UploadMediaUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo, mediaRepo: mediaRepo, storage: storage}
}

// Execute validates the target and content type, saves the file to storage,
// then records the resulting URL. Storage write happens outside any DB
// transaction (external I/O, same reasoning as the WooCommerce calls in
// application/channels) — a failed Attach after a successful Save leaves an
// orphaned file, which is an acceptable, cheap-to-clean-up tradeoff against
// holding a transaction open across a file write.
func (uc *UploadMediaUseCase) Execute(ctx context.Context, in UploadMediaInput) (*pmsDomain.Media, error) {
	if in.ProductID == nil && in.VariantID == nil {
		return nil, fmt.Errorf("uploadMedia: %w", pmsDomain.ErrMediaTargetRequired)
	}
	if !allowedImageContentTypes[in.ContentType] {
		return nil, fmt.Errorf("uploadMedia: %w", pmsDomain.ErrUnsupportedMediaType)
	}

	var tenantID string
	var position int
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if in.ProductID != nil {
			if _, err := uc.productRepo.FindByID(ctx, *in.ProductID); err != nil {
				return err
			}
			existing, err := uc.mediaRepo.ListByProduct(ctx, *in.ProductID)
			if err != nil {
				return err
			}
			position = len(existing)
		} else {
			if _, err := uc.variantRepo.FindByID(ctx, *in.VariantID); err != nil {
				return err
			}
		}
		tenantID = tenant.IDFromContext(ctx)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("uploadMedia: %w", err)
	}

	url, err := uc.storage.Save(ctx, tenantID, in.Filename, in.Data)
	if err != nil {
		return nil, fmt.Errorf("uploadMedia: %w", err)
	}

	var media pmsDomain.Media
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		media = pmsDomain.Media{
			TenantID:  tenant.IDFromContext(ctx),
			ProductID: in.ProductID,
			VariantID: in.VariantID,
			URL:       url,
			Type:      pmsDomain.MediaTypeImage,
			Position:  position,
			AltText:   in.AltText,
			CreatedAt: time.Now(),
		}
		return uc.mediaRepo.Attach(ctx, &media)
	})
	if err != nil {
		return nil, fmt.Errorf("uploadMedia: %w", err)
	}
	return &media, nil
}
