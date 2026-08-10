package pms

import "time"

type SKUStatus string

const (
	SKUStatusActive   SKUStatus = "active"
	SKUStatusInactive SKUStatus = "inactive"
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
	// TracksWeight marks this SKU as a weight pool: its stock quantity is
	// grams, not whole units (e.g. a 20kg bulk lot booked as a single
	// stock-in of 20000). Purely a display/input hint plus a guard for
	// PortionFromBulkUseCase -- the ledger itself stores a plain int64
	// regardless.
	TracksWeight bool
	// PortionedFromSKUID is nil unless this SKU is a sellable item peeled
	// off a TracksWeight bulk SKU's pool (e.g. a "378g pack" portioned
	// from a "Carne Seca — bulk" SKU). Unlike CaseOfSKUID (a fixed 1:1
	// N-multiplier), many SKUs can share the same PortionedFromSKUID,
	// each consuming its own actual weight -- see
	// inventory.PortionFromBulkUseCase. Depth-limited to one level: the
	// referenced SKU must itself have TracksWeight=true and cannot itself
	// have a PortionedFromSKUID (see ErrBulkSKUCannotBePortioned).
	PortionedFromSKUID *string
	Status             SKUStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
