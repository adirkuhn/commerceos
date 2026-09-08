package coupons

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListCouponsUseCase struct {
	db   *gorm.DB
	repo couponsDomain.Repository
}

func NewListCouponsUseCase(db *gorm.DB, repo couponsDomain.Repository) *ListCouponsUseCase {
	return &ListCouponsUseCase{db: db, repo: repo}
}

func (uc *ListCouponsUseCase) Execute(ctx context.Context) ([]couponsDomain.Coupon, error) {
	var coupons []couponsDomain.Coupon
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		coupons, err = uc.repo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listCoupons: %w", err)
	}
	return coupons, nil
}
