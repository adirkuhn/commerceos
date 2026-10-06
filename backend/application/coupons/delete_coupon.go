package coupons

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteCouponUseCase struct {
	db     *gorm.DB
	repo   couponsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewDeleteCouponUseCase(db *gorm.DB, repo couponsDomain.Repository, outbox eventrecord.Enqueuer) *DeleteCouponUseCase {
	return &DeleteCouponUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteCouponUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CouponDeleted, id)
	})
	if err != nil {
		return fmt.Errorf("deleteCoupon: %w", err)
	}
	return nil
}
