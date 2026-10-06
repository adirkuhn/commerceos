package reviews

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteReviewUseCase struct {
	db   *gorm.DB
	repo   reviewsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewDeleteReviewUseCase(db *gorm.DB, repo reviewsDomain.Repository, outbox eventrecord.Enqueuer) *DeleteReviewUseCase {
	return &DeleteReviewUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteReviewUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ReviewDeleted, id)
	})
	if err != nil {
		return fmt.Errorf("deleteReview: %w", err)
	}
	return nil
}
