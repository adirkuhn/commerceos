package reviews_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appreviews "github.com/adirkuhn/commerceos/application/reviews"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
)

func TestListReviews_returnsEveryReview(t *testing.T) {
	repo := &fakeReviewRepo{all: []reviewsDomain.Review{{ID: "review-1"}, {ID: "review-2"}}}
	uc := appreviews.NewListReviewsUseCase(newMockGormDB(t, 1), repo)

	reviews, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, reviews, 2)
}
