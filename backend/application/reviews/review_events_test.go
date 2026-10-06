package reviews_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appreviews "github.com/adirkuhn/commerceos/application/reviews"
)

func TestDeleteReview_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appreviews.NewDeleteReviewUseCase(newMockGormDB(t, 1), &fakeReviewRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "review-1"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "reviews.review.deleted", e.Subject)
	assert.Equal(t, "review-1", e.Payload["entity_id"])
}

func TestDeleteReview_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	uc := appreviews.NewDeleteReviewUseCase(newMockGormDB(t, 1), &fakeReviewRepo{}, &eventrecordtest.Outbox{Err: boom})

	assert.ErrorIs(t, uc.Execute(context.Background(), "review-1"), boom)
}
