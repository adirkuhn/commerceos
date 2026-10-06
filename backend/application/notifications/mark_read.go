package notifications

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type MarkNotificationReadUseCase struct {
	db     *gorm.DB
	repo   notificationsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewMarkNotificationReadUseCase(db *gorm.DB, repo notificationsDomain.Repository, outbox eventrecord.Enqueuer) *MarkNotificationReadUseCase {
	return &MarkNotificationReadUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *MarkNotificationReadUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.MarkRead(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.NotificationRead, id)
	})
	if err != nil {
		return fmt.Errorf("markNotificationRead: %w", err)
	}
	return nil
}

type MarkAllNotificationsReadUseCase struct {
	db     *gorm.DB
	repo   notificationsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewMarkAllNotificationsReadUseCase(db *gorm.DB, repo notificationsDomain.Repository, outbox eventrecord.Enqueuer) *MarkAllNotificationsReadUseCase {
	return &MarkAllNotificationsReadUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *MarkAllNotificationsReadUseCase) Execute(ctx context.Context) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.MarkAllRead(ctx); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.NotificationsAllRead, tenant.IDFromContext(ctx))
	})
	if err != nil {
		return fmt.Errorf("markAllNotificationsRead: %w", err)
	}
	return nil
}
