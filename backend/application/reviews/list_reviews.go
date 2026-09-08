// Package reviews holds the admin-only moderation use cases over shopper-
// submitted reviews (see domain/reviews) -- submitting/listing a product's
// reviews from the storefront side lives in application/storefront instead,
// same split as coupons (domain/coupons vs. application/storefront's
// ApplyCouponUseCase).
package reviews

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListReviewsUseCase struct {
	db   *gorm.DB
	repo reviewsDomain.Repository
}

func NewListReviewsUseCase(db *gorm.DB, repo reviewsDomain.Repository) *ListReviewsUseCase {
	return &ListReviewsUseCase{db: db, repo: repo}
}

func (uc *ListReviewsUseCase) Execute(ctx context.Context) ([]reviewsDomain.Review, error) {
	var reviews []reviewsDomain.Review
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		reviews, err = uc.repo.ListAll(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listReviews: %w", err)
	}
	return reviews, nil
}
