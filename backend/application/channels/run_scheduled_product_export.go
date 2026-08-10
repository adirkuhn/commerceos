package channels

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// runScheduledProductExportLimit mirrors list_products.go's fixed,
// generous page size — real server-side pagination isn't needed here
// either.
const runScheduledProductExportLimit = 200

// productPusher is the one method this use case needs from
// PushProductToWooUseCase (ISP) — defined at point of use so tests don't
// need the full real use case (which itself needs a WooClient, stock
// getter, connection repo, etc).
type productPusher interface {
	Execute(ctx context.Context, in PushProductToWooInput) ([]channelsDomain.ProductChannelListing, error)
}

type RunScheduledProductExportInput struct {
	ConnectionID string
}

type ProductExportResult struct {
	Pushed  int
	Skipped int // already linked to this connection, or has variants (unsupported)
	Errors  []string
}

// RunScheduledProductExportUseCase is the orchestrator cmd/scheduler calls
// for a products/export schedule: finds every eligible product not yet
// linked to this connection and pushes each one via
// PushProductToWooUseCase. Partial-failure tolerant, same as
// ImportWooProductsUseCase — one bad product never blocks the rest.
type RunScheduledProductExportUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	listingRepo channelsDomain.ProductChannelListingRepository
	pusher      productPusher
}

func NewRunScheduledProductExportUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	pusher productPusher,
) *RunScheduledProductExportUseCase {
	return &RunScheduledProductExportUseCase{db: db, productRepo: productRepo, listingRepo: listingRepo, pusher: pusher}
}

func (uc *RunScheduledProductExportUseCase) Execute(ctx context.Context, in RunScheduledProductExportInput) (*ProductExportResult, error) {
	var products []pmsDomain.Product
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, _, err := uc.productRepo.List(ctx, pmsDomain.ProductFilter{Limit: runScheduledProductExportLimit})
		if err != nil {
			return err
		}
		products = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("runScheduledProductExport: %w", err)
	}

	result := &ProductExportResult{}
	for _, p := range products {
		uc.exportOne(ctx, in, p, result)
	}
	return result, nil
}

func (uc *RunScheduledProductExportUseCase) exportOne(ctx context.Context, in RunScheduledProductExportInput, p pmsDomain.Product, result *ProductExportResult) {
	if p.HasVariants {
		result.Skipped++
		return
	}

	var alreadyLinked bool
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		_, err := uc.listingRepo.FindByProductAndConnection(ctx, p.ID, in.ConnectionID)
		if err == nil {
			alreadyLinked = true
			return nil
		}
		if errors.Is(err, channelsDomain.ErrListingNotFound) {
			return nil
		}
		return err
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", p.ID, err))
		return
	}
	if alreadyLinked {
		result.Skipped++
		return
	}

	if _, err := uc.pusher.Execute(ctx, PushProductToWooInput{ProductID: p.ID, ConnectionID: in.ConnectionID}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", p.ID, err))
		return
	}
	result.Pushed++
}
