package notifications

import (
	"context"
	"fmt"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type MarkNotificationReadUseCase struct {
	db   *gorm.DB
	repo notificationsDomain.Repository
}

func NewMarkNotificationReadUseCase(db *gorm.DB, repo notificationsDomain.Repository) *MarkNotificationReadUseCase {
	return &MarkNotificationReadUseCase{db: db, repo: repo}
}

func (uc *MarkNotificationReadUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.MarkRead(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("markNotificationRead: %w", err)
	}
	return nil
}

type MarkAllNotificationsReadUseCase struct {
	db   *gorm.DB
	repo notificationsDomain.Repository
}

func NewMarkAllNotificationsReadUseCase(db *gorm.DB, repo notificationsDomain.Repository) *MarkAllNotificationsReadUseCase {
	return &MarkAllNotificationsReadUseCase{db: db, repo: repo}
}

func (uc *MarkAllNotificationsReadUseCase) Execute(ctx context.Context) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.MarkAllRead(ctx)
	})
	if err != nil {
		return fmt.Errorf("markAllNotificationsRead: %w", err)
	}
	return nil
}
