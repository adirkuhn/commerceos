package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

type fakeReviewRepo struct {
	reviewsDomain.Repository
	created  *reviewsDomain.Review
	verified bool
}

func (f *fakeReviewRepo) Create(_ context.Context, r *reviewsDomain.Review) error {
	r.ID = "review-1"
	f.created = r
	return nil
}

func (f *fakeReviewRepo) HasVerifiedPurchase(_ context.Context, _, _, _ string) (bool, error) {
	return f.verified, nil
}

func activeEnrolledProduct() (*fakeChannelRepo, *fakeProductRepo, *fakeOverrideRepo) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	productRepo := &fakeProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1", Name: "Widget", Status: pmsDomain.ProductStatusActive}}}
	overrideRepo := &fakeOverrideRepo{}
	return channelRepo, productRepo, overrideRepo
}

func TestSubmitReview_createsAnUnverifiedReview(t *testing.T) {
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	reviewRepo := &fakeReviewRepo{}
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 1), channelRepo, overrideRepo, productRepo, reviewRepo)

	review, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1",
		CustomerName: "Ada", CustomerEmail: "ada@example.com", Rating: 5, Title: "Great", Body: "Loved it",
	})

	require.NoError(t, err)
	require.False(t, review.Verified)
	require.Equal(t, "review-1", review.ID)
	require.NotNil(t, reviewRepo.created)
}

func TestSubmitReview_setsVerifiedWhenPurchaseConfirmed(t *testing.T) {
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	reviewRepo := &fakeReviewRepo{verified: true}
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 1), channelRepo, overrideRepo, productRepo, reviewRepo)

	review, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1",
		CustomerName: "Ada", CustomerEmail: "ada@example.com", Rating: 4, Body: "Good",
	})

	require.NoError(t, err)
	require.True(t, review.Verified)
}

func TestSubmitReview_rejectsInvalidRating(t *testing.T) {
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 0), channelRepo, overrideRepo, productRepo, &fakeReviewRepo{})

	_, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1",
		CustomerName: "Ada", CustomerEmail: "ada@example.com", Rating: 6, Body: "Good",
	})

	require.ErrorIs(t, err, reviewsDomain.ErrInvalidRating)
}

func TestSubmitReview_rejectsEmptyBody(t *testing.T) {
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 0), channelRepo, overrideRepo, productRepo, &fakeReviewRepo{})

	_, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1",
		CustomerName: "Ada", CustomerEmail: "ada@example.com", Rating: 3, Body: "   ",
	})

	require.ErrorIs(t, err, reviewsDomain.ErrReviewBodyRequired)
}

func TestSubmitReview_rejectsMissingNameAndEmail(t *testing.T) {
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 0), channelRepo, overrideRepo, productRepo, &fakeReviewRepo{})

	_, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1", Rating: 3, Body: "Good",
	})
	require.ErrorIs(t, err, reviewsDomain.ErrCustomerNameRequired)

	_, err = uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1", CustomerName: "Ada", Rating: 3, Body: "Good",
	})
	require.ErrorIs(t, err, reviewsDomain.ErrCustomerEmailRequired)
}

func TestSubmitReview_rejectsUnavailableProduct(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	productRepo := &fakeProductRepo{}
	overrideRepo := &fakeOverrideRepo{}
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 1), channelRepo, overrideRepo, productRepo, &fakeReviewRepo{})

	_, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "missing-product",
		CustomerName: "Ada", CustomerEmail: "ada@example.com", Rating: 3, Body: "Good",
	})

	require.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}
