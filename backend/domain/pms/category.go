package pms

import "time"

type Category struct {
	ID       string
	TenantID string
	ParentID *string
	Name     string
	Slug     string
	Position int
	// TargetMarkupBasisPoints overrides the tenant's default target markup
	// (settings.TenantSettings.DefaultTargetMarkupBasisPoints) for every
	// product in this category, when set. Basis points rather than a float,
	// same reasoning as pms.SKU.PriceMinorUnits avoiding float money:
	// 3000 = 30% markup. No upper bound (unlike margin). Nil means "use the
	// tenant default" -- see application/pms's
	// EffectiveTargetMarkupBasisPoints.
	TargetMarkupBasisPoints *int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
