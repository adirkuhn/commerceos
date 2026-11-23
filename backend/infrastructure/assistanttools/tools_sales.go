package assistanttools

import (
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// salesTools: shipments, customers and per-warehouse stock.
func salesTools(db *gorm.DB) []assistant.Tool { return nil }
