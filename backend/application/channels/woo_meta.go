package channels

import (
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// toWooMetaData surfaces which batch a synced quantity came from as Woo
// custom fields. Always returns both keys, even when b is nil (blank
// values) -- omitting the keys instead would leave a stale lot_number/
// expiry_date on the Woo side from an earlier sync where the SKU did
// still qualify for StockLevel.SourceBatch (see its doc comment), since
// nothing else ever clears them. Sending blanks on every sync means each
// push is a full, authoritative overwrite of what CommerceOS currently
// knows -- never partial.
func toWooMetaData(b *invapp.BatchSyncInfo) []integrationsDomain.WooMetaData {
	lot, expiry := "", ""
	if b != nil {
		lot = b.LotNumber
		expiry = b.ExpiryDate.Format("2006-01-02")
	}
	return []integrationsDomain.WooMetaData{
		{Key: "lot_number", Value: lot},
		{Key: "expiry_date", Value: expiry},
	}
}
