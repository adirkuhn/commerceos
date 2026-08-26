package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListSalesOrderActivitiesUseCase powers the order detail page's activity
// timeline.
type ListSalesOrderActivitiesUseCase struct {
	db           *gorm.DB
	activityRepo omsDomain.SalesOrderActivityRepository
}

func NewListSalesOrderActivitiesUseCase(db *gorm.DB, activityRepo omsDomain.SalesOrderActivityRepository) *ListSalesOrderActivitiesUseCase {
	return &ListSalesOrderActivitiesUseCase{db: db, activityRepo: activityRepo}
}

func (uc *ListSalesOrderActivitiesUseCase) Execute(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderActivity, error) {
	var activities []omsDomain.SalesOrderActivity
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		activities, err = uc.activityRepo.ListBySalesOrder(ctx, salesOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSalesOrderActivities: %w", err)
	}
	return activities, nil
}

// ListUserActivitiesUseCase powers the Users page's per-user activity
// panel -- the same log, sliced by actor instead of by order.
type ListUserActivitiesUseCase struct {
	db           *gorm.DB
	activityRepo omsDomain.SalesOrderActivityRepository
}

func NewListUserActivitiesUseCase(db *gorm.DB, activityRepo omsDomain.SalesOrderActivityRepository) *ListUserActivitiesUseCase {
	return &ListUserActivitiesUseCase{db: db, activityRepo: activityRepo}
}

func (uc *ListUserActivitiesUseCase) Execute(ctx context.Context, userID string) ([]omsDomain.SalesOrderActivity, error) {
	var activities []omsDomain.SalesOrderActivity
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		activities, err = uc.activityRepo.ListByUser(ctx, userID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listUserActivities: %w", err)
	}
	return activities, nil
}
