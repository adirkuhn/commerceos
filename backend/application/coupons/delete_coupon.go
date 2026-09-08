package coupons

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteCouponUseCase struct {
	db   *gorm.DB
	repo couponsDomain.Repository
}

func NewDeleteCouponUseCase(db *gorm.DB, repo couponsDomain.Repository) *DeleteCouponUseCase {
	return &DeleteCouponUseCase{db: db, repo: repo}
}

func (uc *DeleteCouponUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteCoupon: %w", err)
	}
	return nil
}
