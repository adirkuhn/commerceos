package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListReviewsInput struct {
	SalesChannelID string
	ProductID      string
	Limit          int
	Offset         int
}

type ReviewsResult struct {
	Reviews []reviewsDomain.Review
	Summary reviewsDomain.Summary
}

type ListReviewsUseCase struct {
	db         *gorm.DB
	reviewRepo reviewsDomain.Repository
}

func NewListReviewsUseCase(db *gorm.DB, reviewRepo reviewsDomain.Repository) *ListReviewsUseCase {
	return &ListReviewsUseCase{db: db, reviewRepo: reviewRepo}
}

// Execute deliberately does not check product availability/enrollment
// (unlike SubmitReviewUseCase) -- a product that's been unenrolled or
// deactivated since a review was written should still show its existing
// reviews if the PDP itself is somehow still reachable, and an empty
// review list for a bad product/channel id is indistinguishable from a
// real "no reviews yet" product either way.
func (uc *ListReviewsUseCase) Execute(ctx context.Context, in ListReviewsInput) (*ReviewsResult, error) {
	var result ReviewsResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		reviews, err := uc.reviewRepo.ListByProduct(ctx, reviewsDomain.ListFilter{
			SalesChannelID: in.SalesChannelID, ProductID: in.ProductID, Limit: in.Limit, Offset: in.Offset,
		})
		if err != nil {
			return err
		}
		summary, err := uc.reviewRepo.SummaryByProduct(ctx, in.SalesChannelID, in.ProductID)
		if err != nil {
			return err
		}
		result = ReviewsResult{Reviews: reviews, Summary: summary}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listReviews: %w", err)
	}
	return &result, nil
}
