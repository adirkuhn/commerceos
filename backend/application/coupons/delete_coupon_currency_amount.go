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

type DeleteCouponCurrencyAmountUseCase struct {
	db     *gorm.DB
	repo   couponsDomain.CouponCurrencyAmountRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteCouponCurrencyAmountUseCase(db *gorm.DB, repo couponsDomain.CouponCurrencyAmountRepository, outbox eventrecord.Enqueuer) *DeleteCouponCurrencyAmountUseCase {
	return &DeleteCouponCurrencyAmountUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteCouponCurrencyAmountUseCase) Execute(ctx context.Context, couponID, currencyCode string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, couponID, currencyCode); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CouponCurrencyAmountDeleted, couponID, eventrecord.Data{"currency_code": currencyCode})
	})
	if err != nil {
		return fmt.Errorf("deleteCouponCurrencyAmount: %w", err)
	}
	return nil
}
