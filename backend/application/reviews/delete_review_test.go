package reviews_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appreviews "github.com/adirkuhn/commerceos/application/reviews"
)

func TestDeleteReview_deletesByID(t *testing.T) {
	repo := &fakeReviewRepo{}
	uc := appreviews.NewDeleteReviewUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "review-1")

	require.NoError(t, err)
	require.Equal(t, "review-1", repo.deletedID)
}

func TestDeleteReview_propagatesRepoError(t *testing.T) {
	repo := &fakeReviewRepo{deleteErr: errors.New("boom")}
	uc := appreviews.NewDeleteReviewUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "review-1")

	require.Error(t, err)
}
