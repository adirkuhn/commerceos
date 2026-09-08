package reviews

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteReviewUseCase struct {
	db   *gorm.DB
	repo reviewsDomain.Repository
}

func NewDeleteReviewUseCase(db *gorm.DB, repo reviewsDomain.Repository) *DeleteReviewUseCase {
	return &DeleteReviewUseCase{db: db, repo: repo}
}

func (uc *DeleteReviewUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteReview: %w", err)
	}
	return nil
}
