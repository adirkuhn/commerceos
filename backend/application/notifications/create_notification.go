package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateNotificationInput struct {
	Type    notificationsDomain.Type
	Title   string
	Message string
	Link    string
}

type CreateNotificationUseCase struct {
	db     *gorm.DB
	repo   notificationsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewCreateNotificationUseCase(db *gorm.DB, repo notificationsDomain.Repository, outbox eventrecord.Enqueuer) *CreateNotificationUseCase {
	return &CreateNotificationUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *CreateNotificationUseCase) Execute(ctx context.Context, in CreateNotificationInput) (*notificationsDomain.Notification, error) {
	var n notificationsDomain.Notification
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		n = notificationsDomain.Notification{
			TenantID:  tenant.IDFromContext(ctx),
			Type:      in.Type,
			Title:     in.Title,
			Message:   in.Message,
			Link:      in.Link,
			CreatedAt: time.Now(),
		}
		if err := uc.repo.Create(ctx, &n); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.NotificationCreated, n.ID, eventrecord.Data{
			"type": string(n.Type), "title": n.Title, "link": n.Link,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createNotification: %w", err)
	}
	return &n, nil
}
