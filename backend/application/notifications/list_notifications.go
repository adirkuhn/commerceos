package notifications

import (
	"context"
	"fmt"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const defaultNotificationsLimit = 50

type ListNotificationsUseCase struct {
	db   *gorm.DB
	repo notificationsDomain.Repository
}

func NewListNotificationsUseCase(db *gorm.DB, repo notificationsDomain.Repository) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{db: db, repo: repo}
}

func (uc *ListNotificationsUseCase) Execute(ctx context.Context) ([]notificationsDomain.Notification, error) {
	var out []notificationsDomain.Notification
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.repo.List(ctx, defaultNotificationsLimit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listNotifications: %w", err)
	}
	return out, nil
}
