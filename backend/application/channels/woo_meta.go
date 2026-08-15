package channels

import (
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// toWooMetaData surfaces which batch a synced quantity came from as Woo
// custom fields -- nil when the stock level isn't attributable to one
// specific lot (see StockLevel.SourceBatch's doc comment).
func toWooMetaData(b *invapp.BatchSyncInfo) []integrationsDomain.WooMetaData {
	if b == nil {
		return nil
	}
	return []integrationsDomain.WooMetaData{
		{Key: "lot_number", Value: b.LotNumber},
		{Key: "expiry_date", Value: b.ExpiryDate.Format("2006-01-02")},
	}
}
