// Package storefront implements the internal-storefront checkout use
// cases: cart create/read/update and checkout, all scoped to one specific
// channels.SalesChannel (see domain/storefront.Cart's doc comment for why
// this differs from application/whatsapp's single-channel-per-tenant
// assumption). Deliberately its own package with its own small local
// copies of shared patterns (channel resolution, payment-link generation)
// rather than importing application/whatsapp or application/oms directly
// for them -- same "no cross-application-package coupling" convention
// application/oms.PaymentRefundClientResolver's doc comment documents.
package storefront

import (
	"context"
	"errors"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

// catalogEntry is one SKU code's current, checkout-relevant catalog state
// -- what UpdateCartItemsUseCase needs to build a fresh CartItem snapshot.
// Deliberately carries no stock figure: StockLevelGetter.Execute opens its
// own transaction (see its doc comment), so callers fetch stock as a
// separate step after the transaction lookupCatalogEntry runs in has
// closed -- same "compose across transactions, never nest tenant.WithTenant
// calls" convention application/whatsapp's placeOrder already follows for
// this exact use case.
type catalogEntry struct {
	SKUID               string
	ProductName         string
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	CurrencyCode        string
	TaxClassID          *string
}

// StockLevelGetter is *application/inventory.GetSKUStockLevelUseCase
// narrowed to point of use (ISP), same convention as
// application/whatsapp.stockLevelGetter. Exported (unlike that one) because
// every use case in this package needing stock is a distinct caller of it,
// not a single shared field.
type StockLevelGetter interface {
	Execute(ctx context.Context, skuID string) (*appinventory.StockLevel, error)
}

// lookupCatalogEntry resolves one SKU code against the channel's own
// enrollment rule, returning the exact same error for "no such SKU" and
// "SKU exists but isn't enrolled in this channel" -- same reasoning as
// application/whatsapp.ErrProductNotFound: a storefront request must never
// be able to use this to probe which SKU codes exist beyond what's
// actually enrolled here. Must be called from inside a tenant.WithTenant
// block (it takes no db of its own) -- callers fetch stock separately,
// once that block has closed.
func lookupCatalogEntry(
	ctx context.Context,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	channel channelsDomain.SalesChannel,
	skuCode string,
) (*catalogEntry, error) {
	sku, err := skuRepo.FindByCode(ctx, skuCode)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return nil, storefrontDomain.ErrProductNotAvailable
		}
		return nil, err
	}
	variant, err := variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrVariantNotFound) {
			return nil, storefrontDomain.ErrProductNotAvailable
		}
		return nil, err
	}
	product, err := productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrProductNotFound) {
			return nil, storefrontDomain.ErrProductNotAvailable
		}
		return nil, err
	}
	override, err := overrideRepo.FindByChannelAndProduct(ctx, channel.ID, product.ID)
	if err != nil {
		return nil, err
	}
	if !channel.IsProductEnrolled(product.CategoryID, product.Tags, override) {
		return nil, storefrontDomain.ErrProductNotAvailable
	}
	// Both prices go through the channel's own discount (e.g. a B2B
	// channel's wholesale rate) before anything else ever sees them --
	// snapshotted onto the CartItem at this point, same as every other
	// price field here (see catalogEntry's own doc comment).
	price := channel.EffectivePriceMinorUnits(sku.PriceMinorUnits)
	var salePrice *int64
	if sku.SalePriceMinorUnits != nil {
		discounted := channel.EffectivePriceMinorUnits(*sku.SalePriceMinorUnits)
		salePrice = &discounted
	}
	return &catalogEntry{
		SKUID: sku.ID, ProductName: product.Name, PriceMinorUnits: price,
		SalePriceMinorUnits: salePrice, CurrencyCode: sku.CurrencyCode, TaxClassID: sku.TaxClassID,
	}, nil
}

// EffectivePriceMinorUnits is SalePriceMinorUnits when a sale is active,
// else PriceMinorUnits -- same precedence application/whatsapp's
// handleAddToCart applies.
func (e catalogEntry) EffectivePriceMinorUnits() int64 {
	if e.SalePriceMinorUnits != nil {
		return *e.SalePriceMinorUnits
	}
	return e.PriceMinorUnits
}

// findInternalChannel resolves a storefront key's sales channel by ID and
// verifies it's an active internal channel -- every cart use case's entry
// point. Unlike application/whatsapp.findActiveWhatsAppChannel (which
// scans for "the" WhatsApp channel, assuming at most one per tenant), a
// storefront key always names its own specific channel, since a tenant can
// run more than one internal channel (see the package doc comment).
func findInternalChannel(ctx context.Context, channelRepo channelsDomain.SalesChannelRepository, salesChannelID string) (*channelsDomain.SalesChannel, error) {
	channel, err := channelRepo.FindByID(ctx, salesChannelID)
	if err != nil {
		if errors.Is(err, channelsDomain.ErrSalesChannelNotFound) {
			return nil, storefrontDomain.ErrNoInternalChannel
		}
		return nil, err
	}
	if channel.Type != channelsDomain.ChannelTypeInternal || !channel.Active {
		return nil, storefrontDomain.ErrNoInternalChannel
	}
	return channel, nil
}
