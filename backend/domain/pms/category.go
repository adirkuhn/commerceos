package pms

import "time"

type Category struct {
	ID       string
	TenantID string
	ParentID *string
	Name     string
	Slug     string
	Position int
	// TargetMarginBasisPoints overrides the tenant's default target margin
	// (settings.TenantSettings.DefaultTargetMarginBasisPoints) for every
	// product in this category, when set. Basis points (0-9999, i.e. a
	// margin strictly under 100%) rather than a float, same reasoning as
	// pms.SKU.PriceMinorUnits avoiding float money: 3000 = 30%. Nil means
	// "use the tenant default" -- see application/pms's
	// EffectiveTargetMarginBasisPoints.
	TargetMarginBasisPoints *int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
