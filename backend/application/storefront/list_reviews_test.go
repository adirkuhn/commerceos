package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
)

type fakeListReviewsRepo struct {
	reviewsDomain.Repository
	reviews   []reviewsDomain.Review
	summary   reviewsDomain.Summary
	gotFilter reviewsDomain.ListFilter
}

func (f *fakeListReviewsRepo) ListByProduct(_ context.Context, filter reviewsDomain.ListFilter) ([]reviewsDomain.Review, error) {
	f.gotFilter = filter
	return f.reviews, nil
}

func (f *fakeListReviewsRepo) SummaryByProduct(_ context.Context, _, _ string) (reviewsDomain.Summary, error) {
	return f.summary, nil
}

func TestListReviews_returnsReviewsAndSummary(t *testing.T) {
	repo := &fakeListReviewsRepo{
		reviews: []reviewsDomain.Review{{ID: "review-1", Rating: 5}},
		summary: reviewsDomain.Summary{Average: 4.5, Count: 2},
	}
	uc := appstorefront.NewListReviewsUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), appstorefront.ListReviewsInput{SalesChannelID: "channel-1", ProductID: "product-1"})

	require.NoError(t, err)
	require.Len(t, result.Reviews, 1)
	require.Equal(t, 4.5, result.Summary.Average)
	require.Equal(t, 2, result.Summary.Count)
	require.Equal(t, "channel-1", repo.gotFilter.SalesChannelID)
	require.Equal(t, "product-1", repo.gotFilter.ProductID)
}
