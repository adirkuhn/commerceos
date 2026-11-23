package assistanttools

import (
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// operationsTools: suppliers, purchase orders, batches and expiry,
// integration connections and sync logs.
func operationsTools(db *gorm.DB) []assistant.Tool { return nil }
