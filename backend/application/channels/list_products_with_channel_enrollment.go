package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ProductEnrollment pairs a product with its live-computed enrollment
// status for one channel -- Enrolled is never stored (see
// SalesChannel.IsProductEnrolled), it's recalculated on every call so a
// channel's rule change is reflected immediately with no backfill.
type ProductEnrollment struct {
	Product  pmsDomain.Product
	Enrolled bool
}

type ListProductsWithChannelEnrollmentInput struct {
	ChannelID string
	Search    *string
	Limit     int
	Offset    int
}

// ListProductsWithChannelEnrollmentUseCase powers the Sales Channel
// product-picker UI: the tenant's products, paginated/searchable, each
// tagged with whether it currently belongs on this channel.
type ListProductsWithChannelEnrollmentUseCase struct {
	db           *gorm.DB
	productRepo  pmsDomain.ProductRepository
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
}

func NewListProductsWithChannelEnrollmentUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
) *ListProductsWithChannelEnrollmentUseCase {
	return &ListProductsWithChannelEnrollmentUseCase{db: db, productRepo: productRepo, channelRepo: channelRepo, overrideRepo: overrideRepo}
}

func (uc *ListProductsWithChannelEnrollmentUseCase) Execute(ctx context.Context, in ListProductsWithChannelEnrollmentInput) ([]ProductEnrollment, int64, error) {
	var result []ProductEnrollment
	var total int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := uc.channelRepo.FindByID(ctx, in.ChannelID)
		if err != nil {
			return err
		}

		products, count, err := uc.productRepo.List(ctx, pmsDomain.ProductFilter{Search: in.Search, Limit: in.Limit, Offset: in.Offset})
		if err != nil {
			return err
		}
		total = count

		overrides, err := uc.overrideRepo.ListByChannel(ctx, in.ChannelID)
		if err != nil {
			return err
		}
		overrideByProduct := make(map[string]*channelsDomain.SalesChannelProductOverride, len(overrides))
		for i := range overrides {
			overrideByProduct[overrides[i].ProductID] = &overrides[i]
		}

		result = make([]ProductEnrollment, len(products))
		for i, p := range products {
			result[i] = ProductEnrollment{Product: p, Enrolled: channel.IsProductEnrolled(p.CategoryID, p.Tags, overrideByProduct[p.ID])}
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listProductsWithChannelEnrollment: %w", err)
	}
	return result, total, nil
}
