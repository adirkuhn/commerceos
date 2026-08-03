package channels

import "time"

type SyncEntityType string

const (
	SyncEntityProducts   SyncEntityType = "products"
	SyncEntityStock      SyncEntityType = "stock"
	SyncEntityOrders     SyncEntityType = "orders"     // schema-legal, no runner yet
	SyncEntityCategories SyncEntityType = "categories" // schema-legal, no runner yet
)

type SyncDirection string

const (
	SyncDirectionImport SyncDirection = "import"
	SyncDirectionExport SyncDirection = "export"
)

// SyncSchedule is a per-connection, per-entity, per-direction recurring
// sync job configuration — one row per (ConnectionID, EntityType,
// Direction), upserted rather than appended. cmd/scheduler polls
// SyncScheduleRepository.ListDue and dispatches to the matching use case.
type SyncSchedule struct {
	ID              string
	TenantID        string
	ConnectionID    string
	EntityType      SyncEntityType
	Direction       SyncDirection
	Enabled         bool
	IntervalMinutes int
	// DefaultWarehouseID/DefaultCurrency are only required (and only
	// meaningful) for EntityType=products, Direction=import — Woo has no
	// warehouse concept and its product endpoint doesn't return a
	// currency, so ImportWooProductsUseCase needs both supplied
	// per-connection. Unused by every other entity/direction combination.
	DefaultWarehouseID *string
	DefaultCurrency    *string
	LastRunAt          *time.Time
	LastRunStatus      string // "" | "success" | "error"
	LastRunError       string
	NextRunAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
