package storefront

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type SubmitReviewInput struct {
	SalesChannelID string
	ProductID      string
	CustomerName   string
	CustomerEmail  string
	Rating         int
	Title          string
	Body           string
}

type SubmitReviewUseCase struct {
	db           *gorm.DB
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	productRepo  pmsDomain.ProductRepository
	reviewRepo   reviewsDomain.Repository
	outbox       eventrecord.Enqueuer
}

func NewSubmitReviewUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	productRepo pmsDomain.ProductRepository,
	reviewRepo reviewsDomain.Repository,
	outbox eventrecord.Enqueuer,
) *SubmitReviewUseCase {
	return &SubmitReviewUseCase{db: db, channelRepo: channelRepo, overrideRepo: overrideRepo, productRepo: productRepo, reviewRepo: reviewRepo, outbox: outbox}
}

// Execute validates the review, confirms the product is actually available
// on this channel (same ErrProductNotAvailable check as GetProductUseCase --
// a review can't be left on a product the shopper couldn't even see),
// checks for a matching completed order to set Verified, then persists.
func (uc *SubmitReviewUseCase) Execute(ctx context.Context, in SubmitReviewInput) (*reviewsDomain.Review, error) {
	name := strings.TrimSpace(in.CustomerName)
	email := strings.TrimSpace(in.CustomerEmail)
	body := strings.TrimSpace(in.Body)
	if name == "" {
		return nil, reviewsDomain.ErrCustomerNameRequired
	}
	if email == "" {
		return nil, reviewsDomain.ErrCustomerEmailRequired
	}
	if body == "" {
		return nil, reviewsDomain.ErrReviewBodyRequired
	}
	review := reviewsDomain.Review{
		SalesChannelID: in.SalesChannelID, ProductID: in.ProductID,
		CustomerName: name, CustomerEmail: email, Rating: in.Rating, Title: strings.TrimSpace(in.Title), Body: body,
	}
	if !review.ValidRating() {
		return nil, reviewsDomain.ErrInvalidRating
	}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := findInternalChannel(ctx, uc.channelRepo, in.SalesChannelID)
		if err != nil {
			return err
		}
		product, err := uc.productRepo.FindByID(ctx, in.ProductID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrProductNotFound) {
				return storefrontDomain.ErrProductNotAvailable
			}
			return err
		}
		if product.Status != pmsDomain.ProductStatusActive {
			return storefrontDomain.ErrProductNotAvailable
		}
		override, err := uc.overrideRepo.FindByChannelAndProduct(ctx, in.SalesChannelID, in.ProductID)
		if err != nil {
			return err
		}
		if !channel.IsProductEnrolled(product.CategoryID, product.Tags, override) {
			return storefrontDomain.ErrProductNotAvailable
		}

		review.TenantID = channel.TenantID
		verified, err := uc.reviewRepo.HasVerifiedPurchase(ctx, in.SalesChannelID, in.ProductID, email)
		if err != nil {
			return err
		}
		review.Verified = verified
		now := time.Now()
		review.CreatedAt = now
		review.UpdatedAt = now
		if err := uc.reviewRepo.Create(ctx, &review); err != nil {
			return err
		}
		// Reviewer name/email and the review text stay out of the event.
		return eventrecord.Record(ctx, uc.outbox, events.StorefrontReviewSubmitted, review.ID, eventrecord.Data{
			"sales_channel_id": review.SalesChannelID, "product_id": review.ProductID,
			"rating": review.Rating, "verified": review.Verified,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("submitReview: %w", err)
	}
	return &review, nil
}
