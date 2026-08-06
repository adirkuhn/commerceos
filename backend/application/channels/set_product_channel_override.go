package channels

import (
	"context"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type SetProductChannelOverrideInput struct {
	ChannelID string
	ProductID string
	// Included true = force-include ("add"), false = force-exclude
	// ("remove") -- always wins over the channel's default EnrollmentMode
	// rule for this one product. See SalesChannelProductOverride's doc
	// comment.
	Included bool
}

type SetProductChannelOverrideUseCase struct {
	db           *gorm.DB
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	channelRepo  channelsDomain.SalesChannelRepository
	productRepo  pmsDomain.ProductRepository
}

func NewSetProductChannelOverrideUseCase(
	db *gorm.DB,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	productRepo pmsDomain.ProductRepository,
) *SetProductChannelOverrideUseCase {
	return &SetProductChannelOverrideUseCase{db: db, overrideRepo: overrideRepo, channelRepo: channelRepo, productRepo: productRepo}
}

func (uc *SetProductChannelOverrideUseCase) Execute(ctx context.Context, in SetProductChannelOverrideInput) (*channelsDomain.SalesChannelProductOverride, error) {
	var override channelsDomain.SalesChannelProductOverride
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.channelRepo.FindByID(ctx, in.ChannelID); err != nil {
			return err
		}
		if _, err := uc.productRepo.FindByID(ctx, in.ProductID); err != nil {
			return err
		}
		override = channelsDomain.SalesChannelProductOverride{
			TenantID: tenant.IDFromContext(ctx), ChannelID: in.ChannelID, ProductID: in.ProductID,
			Included: in.Included, CreatedAt: time.Now(),
		}
		return uc.overrideRepo.Upsert(ctx, &override)
	})
	if err != nil {
		return nil, fmt.Errorf("setProductChannelOverride: %w", err)
	}
	return &override, nil
}
