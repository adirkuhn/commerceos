package coupons

import "context"

type Repository interface {
	Create(ctx context.Context, c *Coupon) error
	FindByID(ctx context.Context, id string) (*Coupon, error)
	// FindByCode returns nil, nil (not ErrCouponNotFound) when no coupon
	// matches -- callers (application/storefront's checkout/apply-coupon
	// flow) map that to their own not-redeemable error, same "repo returns
	// nil, use case owns the domain error" convention as
	// theme.Repository.FindBySlug.
	FindByCode(ctx context.Context, code string) (*Coupon, error)
	ListByTenant(ctx context.Context) ([]Coupon, error)
	Update(ctx context.Context, c *Coupon) error
	Delete(ctx context.Context, id string) error
	// IncrementTimesUsed is a single atomic UPDATE ... SET times_used =
	// times_used + 1 -- never read-modify-write in application code, so
	// two concurrent checkouts redeeming the same code can't race and
	// silently undercount.
	IncrementTimesUsed(ctx context.Context, id string) error
}
