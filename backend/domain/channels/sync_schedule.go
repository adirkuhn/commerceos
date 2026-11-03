package channels

import "time"

type SyncEntityType string

const (
	// Woo products and stock have no schedule (COM-18): every change
	// reaches the store through events (pms.product.changed,
	// inventory.stock.changed), and stock drift is caught by
	// SyncEntityStockReconciliation. The old "products"/"stock" rows were
	// deleted by migration 243.
	SyncEntityOrders     SyncEntityType = "orders"     // import only -- see cmd/scheduler/dispatch.go's buildRunners doc
	SyncEntityCategories SyncEntityType = "categories" // schema-legal, no runner yet
	// SyncEntityOrdersReconciliation is a second, independent schedule
	// alongside SyncEntityOrders for the same connection -- import only,
	// re-runs ImportWooOrdersUseCase over a fixed rolling window instead of
	// an incremental cursor, to self-heal any order SyncEntityOrders ever
	// missed (whatever the cause). See cmd/scheduler/dispatch.go's
	// reconciliationWindow.
	SyncEntityOrdersReconciliation SyncEntityType = "orders_reconciliation"
	// SyncEntityShipmentReconciliation is the shipping module's own
	// reconciliation schedule (Direction=import, one row per MyParcel
	// connection) -- same "re-scan regardless of LastRunAt, rely on the
	// underlying operation's own idempotency" shape as
	// SyncEntityOrdersReconciliation, just for
	// application/shipping.ReconcileShipmentsUseCase instead of a Woo
	// order import. Reuses this generic per-connection schedule shape
	// rather than a parallel scheduling mechanism, even though shipping
	// isn't itself a "channel."
	SyncEntityShipmentReconciliation SyncEntityType = "shipment_reconciliation"
	// SyncEntityBolOrders/BolProducts/BolStock are Bol's own entity types,
	// distinct from products/stock/orders above -- cmd/scheduler/dispatch.go's
	// runner map has exactly one runner per (entity_type, direction) key, so
	// reusing Woo's keys would silently route a Bol connection's schedule
	// through the Woo-only runner. Same reasoning as
	// SyncEntityShipmentReconciliation vs SyncEntityOrdersReconciliation.
	// BolOrders is import-only (no order push-back, same as Woo's orders);
	// BolProducts/BolStock are both export-only and both dispatch to the
	// same runner (Bol has no stock-only endpoint -- every push sends
	// price+stock together) on two separate schedule rows/intervals.
	SyncEntityBolOrders   SyncEntityType = "bol_orders"
	SyncEntityBolProducts SyncEntityType = "bol_products"
	SyncEntityBolStock    SyncEntityType = "bol_stock"
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
	// DefaultWarehouseID is required (Direction=import) for
	// products/stock/orders -- Woo has no warehouse concept, so every
	// import use case needs one supplied per-connection to attribute
	// stock movements to. DefaultCurrency is additionally required for
	// products only (Woo's product endpoint doesn't return a currency;
	// orders carry their own per-order currency already). Unused by
	// every other entity/direction combination.
	DefaultWarehouseID *string
	DefaultCurrency    *string
	LastRunAt          *time.Time
	LastRunStatus      string // "" | "success" | "error"
	LastRunError       string
	NextRunAt          *time.Time
	// FailingSince is when the current run of consecutive failures began;
	// nil while healthy. FailureAlerted: staff were told about it (see
	// ApplyRunOutcome).
	FailingSince   *time.Time
	FailureAlerted bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
