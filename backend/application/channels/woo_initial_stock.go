package channels

import (
	"context"
	"log"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// wooInitialStock is the stock a physical SKU's Woo product/variation is
// created with. Always managed: Woo treats a product without
// manage_stock as "in stock" with no limit, so creating one unmanaged lets
// shoppers order any quantity until some later stock movement happens to
// push a real count. A failed (or unwired) stock lookup therefore fails
// closed -- quantity 0, out of stock until the next stock sync -- rather
// than open.
type wooInitialStock struct {
	Quantity int
	MetaData []integrationsDomain.WooMetaData
}

func resolveWooInitialStock(ctx context.Context, stockLevel stockLevelGetter, sku pmsDomain.SKU) wooInitialStock {
	if stockLevel == nil {
		return wooInitialStock{}
	}
	level, err := stockLevel.Execute(ctx, sku.ID)
	if err != nil || level == nil {
		log.Printf("wooInitialStock: stock level for SKU %s, creating it out of stock: %v", sku.ID, err)
		return wooInitialStock{}
	}
	return wooInitialStock{Quantity: int(level.Syncable), MetaData: toWooMetaData(level.SourceBatch)}
}

func (s wooInitialStock) applyToProduct(p *integrationsDomain.WooProduct) {
	qty := s.Quantity
	p.ManageStock = true
	p.StockQuantity = &qty
	p.MetaData = s.MetaData
}

func (s wooInitialStock) applyToVariation(v *integrationsDomain.WooVariation) {
	qty := s.Quantity
	v.ManageStock = true
	v.StockQuantity = &qty
	v.MetaData = s.MetaData
}
