package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ErrNoWhatsAppChannel is returned when the tenant has no active
// ChannelTypeWhatsApp SalesChannel -- WhatsApp ordering/lookup was never
// set up, distinct from ErrProductNotFound/ErrProductNotEnrolled below.
var ErrNoWhatsAppChannel = errors.New("whatsapp: no active whatsapp sales channel configured for this tenant")

// ErrProductNotFound covers both "no SKU with this code" and "SKU exists
// but its product isn't enrolled in the WhatsApp channel" -- deliberately
// the same error/message for both (see LookupProductInfoUseCase's doc
// comment), so an inbound message can never be used to probe which SKU
// codes exist in the catalog beyond what's actually WhatsApp-enrolled.
var ErrProductNotFound = errors.New("whatsapp: product not found")

// stockLevelGetter is *application/inventory.GetSKUStockLevelUseCase
// narrowed to point of use (ISP) -- lets this package stay decoupled from
// inventory's own dependencies (settings repo, etc).
type stockLevelGetter interface {
	Execute(ctx context.Context, skuID string) (*appinventory.StockLevel, error)
}

// ProductInfo is the reply-formatting shape LookupProductInfoUseCase
// returns -- HandleInboundMessageUseCase's formatProductInfo renders it as
// text, never exposing internal ids. SKUID is the one exception -- carried
// through for OrderFlowUseCase's cart (a CartItem needs the real SKUID to
// eventually create a SalesOrderLineItem), never rendered into reply text.
type ProductInfo struct {
	SKUID               string
	ProductName         string
	SKUCode             string
	StockQuantity       int64
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	CurrencyCode        string
	// NearestExpiry is nil when the SKU's syncable stock isn't attributed
	// to one specific expiring batch -- see appinventory.StockLevel.SourceBatch's
	// doc comment for exactly when that is.
	NearestExpiry *time.Time
}

// LookupProductInfoUseCase answers a WhatsApp customer's stock/price/expiry
// question for one SKU code, gated by the tenant's WhatsApp SalesChannel
// enrollment rule (domain/channels.SalesChannel.IsProductEnrolled) -- the
// exact same mechanism every other channel (Woo, Bol) uses to decide which
// products it exposes, reused rather than reimplemented (see
// application/channels.ListProductsWithChannelEnrollmentUseCase for the
// precedent).
type LookupProductInfoUseCase struct {
	db           *gorm.DB
	skuRepo      pmsDomain.SKURepository
	variantRepo  pmsDomain.VariantRepository
	productRepo  pmsDomain.ProductRepository
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	stockLevel   stockLevelGetter
}

func NewLookupProductInfoUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	stockLevel stockLevelGetter,
) *LookupProductInfoUseCase {
	return &LookupProductInfoUseCase{
		db: db, skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
		channelRepo: channelRepo, overrideRepo: overrideRepo, stockLevel: stockLevel,
	}
}

func (uc *LookupProductInfoUseCase) Execute(ctx context.Context, skuCode string) (*ProductInfo, error) {
	var info ProductInfo
	var skuID string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := uc.findActiveWhatsAppChannel(ctx)
		if err != nil {
			return err
		}

		sku, err := uc.skuRepo.FindByCode(ctx, skuCode)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				return ErrProductNotFound
			}
			return err
		}
		variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrVariantNotFound) {
				return ErrProductNotFound
			}
			return err
		}
		product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrProductNotFound) {
				return ErrProductNotFound
			}
			return err
		}

		override, err := uc.overrideRepo.FindByChannelAndProduct(ctx, channel.ID, product.ID)
		if err != nil {
			return err
		}
		if !channel.IsProductEnrolled(product.CategoryID, product.Tags, override) {
			return ErrProductNotFound
		}

		skuID = sku.ID
		info = ProductInfo{
			SKUID: sku.ID, ProductName: product.Name, SKUCode: sku.SKUCode,
			PriceMinorUnits: sku.PriceMinorUnits, SalePriceMinorUnits: sku.SalePriceMinorUnits,
			CurrencyCode: sku.CurrencyCode,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("lookupProductInfo: %w", err)
	}

	level, err := uc.stockLevel.Execute(ctx, skuID)
	if err != nil {
		return nil, fmt.Errorf("lookupProductInfo: stock level: %w", err)
	}
	info.StockQuantity = level.Syncable
	if level.SourceBatch != nil {
		expiry := level.SourceBatch.ExpiryDate
		info.NearestExpiry = &expiry
	}
	return &info, nil
}

func (uc *LookupProductInfoUseCase) findActiveWhatsAppChannel(ctx context.Context) (*channelsDomain.SalesChannel, error) {
	channels, err := uc.channelRepo.ListByTenant(ctx)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		if channels[i].Type == channelsDomain.ChannelTypeWhatsApp && channels[i].Active {
			return &channels[i], nil
		}
	}
	return nil, ErrNoWhatsAppChannel
}
