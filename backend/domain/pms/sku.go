package pms

import "time"

type SKUStatus string

const (
	SKUStatusActive   SKUStatus = "active"
	SKUStatusInactive SKUStatus = "inactive"
)

// SKUCostPriceSource says who last set CostPriceMinorUnits.
type SKUCostPriceSource string

const (
	// SKUCostPriceSourceComputed is the default -- the daily cron (see
	// application/pms.RecalculateCostPricesUseCase) owns the value.
	SKUCostPriceSourceComputed SKUCostPriceSource = "computed"
	// SKUCostPriceSourceManual means a human set CostPriceMinorUnits
	// directly (see application/pms.SetManualCostPriceUseCase) -- the cron
	// skips any SKU with this source entirely, leaving the manual value in
	// place until ClearCostPriceOverrideUseCase resets the source back to
	// computed.
	SKUCostPriceSourceManual SKUCostPriceSource = "manual"
	// SKUCostPriceSourceUnknown means this SKU has on-hand quantity with no
	// reliable cost basis -- CostPriceMinorUnits is nil, and a sale against
	// it records no cost_at_sale/COGS (never a fabricated 0, see the MWAC
	// opening-balance migration policy). Resolved automatically the first
	// time a real receipt gives the SKU a cost (see
	// application/pms.ApplyReceiptToMovingAverageCost).
	SKUCostPriceSourceUnknown SKUCostPriceSource = "unknown"
)

// SKUUnitType is how this SKU's stock quantity is denominated. "unit" (the
// default) and "box" are both whole-count stock -- box is purely a
// descriptive/display distinction from unit, carrying no different
// quantity semantics of its own (a case/box relationship to another SKU is
// CaseOfSKUID/UnitsPerCase, not this field). "kilo" means stock quantity is
// grams, not a count -- see SKU.UnitType's own doc comment.
type SKUUnitType string

const (
	SKUUnitTypeUnit SKUUnitType = "unit"
	SKUUnitTypeBox  SKUUnitType = "box"
	SKUUnitTypeKilo SKUUnitType = "kilo"
)

// SKUPricingMode governs how a SKU's sale price is computed at checkout --
// orthogonal to UnitType (which governs how its *stock* is denominated).
// A weight-embedded-barcode packed item (see ScaleCode/
// DecodeWeightEmbeddedBarcode) can be PricingModeFixed (e.g. calabresa,
// kit feijoada -- every package sells for PriceMinorUnits regardless of
// its actual weight) or PricingModePerWeight (e.g. dry meat -- the total
// is PricePerKiloMinorUnits times the scanned package's actual weight).
// Independently, either kind can have UnitType=kilo if the tenant also
// wants stock tracked by weight rather than by package count.
type SKUPricingMode string

const (
	SKUPricingModeFixed     SKUPricingMode = "fixed"
	SKUPricingModePerWeight SKUPricingMode = "per_weight"
)

type SKU struct {
	ID        string
	TenantID  string
	VariantID string
	SKUCode   string
	Barcode   string
	// PriceMinorUnits is the price in the smallest currency unit (e.g. cents for USD).
	PriceMinorUnits int64
	// SalePriceMinorUnits is an optional discounted price, same unit as
	// PriceMinorUnits. Nil means no sale is active. When set it must be
	// <= PriceMinorUnits (see ErrSalePriceExceedsPrice) -- a "sale" that
	// costs more than the regular price isn't one.
	SalePriceMinorUnits *int64
	// SaleEndDate is an optional date after which SalePriceMinorUnits
	// automatically reverts to nil -- the sale is active through this
	// date (inclusive); the scheduler clears it once the date has
	// passed (see application/pms's ExpireSKUSalePriceUseCase). Only
	// meaningful when SalePriceMinorUnits is set (see
	// ErrSaleEndDateRequiresSalePrice).
	SaleEndDate  *time.Time
	CurrencyCode string
	// CostPriceMinorUnits is the replacement cost pricing and margins are
	// judged against: the landed unit cost of the last goods receipt (see
	// CostPriceAfterReceipt), or a manual value (CostPriceSource manual).
	// Nil until a receipt or a human gives this SKU a cost.
	CostPriceMinorUnits *int64
	// InventoryCostMinorUnits is the actual cost of the stock on hand: the
	// moving weighted average of every receipt's landed cost, blended at
	// receipt time (application/pms.ApplyReceiptCostUseCase). COGS fallback,
	// stock value and the cost ledger use this, never CostPriceMinorUnits
	// -- see ActualCostMinorUnits. Kept even while the cost price is manual.
	InventoryCostMinorUnits *int64
	// CostPriceSource says who last set CostPriceMinorUnits -- see
	// SKUCostPriceSource. Defaults to SKUCostPriceSourceComputed.
	CostPriceSource SKUCostPriceSource
	// WeightGrams and the *MM dimensions are nil when unknown — normalized
	// to fixed SI units (like PriceMinorUnits normalizes money) so values
	// from any future channel are comparable without ambiguity.
	WeightGrams *int64
	LengthMM    *int64
	WidthMM     *int64
	HeightMM    *int64
	// IsVirtual marks a digital/non-shippable SKU — never has weight or
	// dimensions set (see ErrVirtualSKUCannotHaveDimensions).
	IsVirtual bool
	// TracksExpiry marks a SKU whose inventory batches carry a
	// meaningful expiry date (e.g. food, cosmetics) — the frontend uses
	// this to decide whether to require/show an expiry date when
	// creating a batch. Purely advisory: a batch can still be created
	// with no expiry regardless of this flag (e.g. general-ledger stock,
	// or a lot whose expiry just isn't known yet).
	TracksExpiry bool
	// CaseOfSKUID/UnitsPerCase are both nil unless this SKU represents a
	// sealed case/box of another SKU (e.g. a 36-unit case of "Rapadurinha
	// de Minas 25g") -- always set together (see
	// ErrCaseOfSKURequiresUnitsPerCase). Depth-limited to one level: the
	// referenced SKU can't itself be a case of yet another SKU (see
	// ErrCaseOfSKUCannotChain) -- keeps stock-split math (see
	// inventory.SplitCaseIntoUnitsUseCase) a single multiplication instead
	// of a walk up an arbitrary chain. This is purely a catalog
	// relationship for later splitting case-stock into unit-stock -- it
	// does not affect how a case SKU is priced, sold, or matched during
	// procurement (see procurement.PurchaseOrderLineItem.UnitsPerPurchasedItem
	// for the separate, unrelated "supplier sells in bulk but we don't
	// stock the bulk unit at all" case).
	CaseOfSKUID  *string
	UnitsPerCase *int64
	// UnitType defaults to SKUUnitTypeUnit. SKUUnitTypeKilo marks this SKU
	// as a weight pool: its stock quantity is grams, not whole units (e.g.
	// a 20kg bulk lot booked as a single stock-in of 20000) -- a guard for
	// PortionFromBulkUseCase, plus a display/input hint. The ledger itself
	// stores a plain int64 regardless of UnitType.
	UnitType SKUUnitType
	// PortionedFromSKUID is nil unless this SKU is a sellable item peeled
	// off a kilo (weight-pool) SKU's pool (e.g. a "378g pack" portioned
	// from a "Carne Seca — bulk" SKU). Unlike CaseOfSKUID (a fixed 1:1
	// N-multiplier), many SKUs can share the same PortionedFromSKUID,
	// each consuming its own actual weight -- see
	// inventory.PortionFromBulkUseCase. Depth-limited to one level: the
	// referenced SKU must itself have UnitType=kilo and cannot itself
	// have a PortionedFromSKUID (see ErrBulkSKUCannotBePortioned).
	PortionedFromSKUID *string
	// ScaleCode is nil unless this SKU is sold via a weight-embedded
	// barcode (see DecodeWeightEmbeddedBarcode) -- the 5-digit code the
	// tenant programs into their scale for this product, distinct from
	// Barcode/BarcodeAlias.Code (which are exact-match, but every printed
	// label for this SKU has a different full barcode since the weight
	// varies per package). Tenant-unique when set.
	ScaleCode *string
	// PricingMode defaults to SKUPricingModeFixed -- see SKUPricingMode's
	// doc comment. Only meaningful for scanning a weight-embedded barcode;
	// a SKU with no ScaleCode is always priced at PriceMinorUnits
	// regardless of this field.
	PricingMode SKUPricingMode
	// PricePerKiloMinorUnits is required (and must be positive) when
	// PricingMode is SKUPricingModePerWeight, and must be nil otherwise
	// (see ErrPricePerKiloRequiredForPerWeight/
	// ErrPricePerKiloOnlyForPerWeight). Same minor-unit convention as
	// PriceMinorUnits, priced per whole kilo.
	PricePerKiloMinorUnits *int64
	// TaxClassID is nil unless this SKU has its own tax.TaxClass assigned
	// -- nil means "use the tenant's default class" (see
	// tax.TaxClass.IsDefault), the common case, so most SKUs never set
	// this at all.
	TaxClassID *string
	// WooTaxClassSlug/WooTaxStatus are informational provenance only, set
	// once by ImportWooProductsUseCase when this SKU came from a
	// WooCommerce store -- never read by tax calculation (that always
	// goes through TaxClassID). "" for WooTaxClassSlug means Woo's own
	// Standard rate. Both nil for a SKU never imported from Woo.
	WooTaxClassSlug *string
	WooTaxStatus    *string
	Status          SKUStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
