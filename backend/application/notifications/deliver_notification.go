package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeliverNotificationUseCase raises a notification the way the tenant
// asked for it: into the admin bell and/or by email, per its category's
// preferences. A drop-in for CreateNotificationUseCase (same input and
// signature) for every producer. The email itself is sent later by a
// consumer of NotificationEmailRequested, never inside this transaction.
type DeliverNotificationUseCase struct {
	db        *gorm.DB
	repo      notificationsDomain.Repository
	prefsRepo notificationsDomain.PreferencesRepository
	outbox    EventEnqueuer
}

func NewDeliverNotificationUseCase(db *gorm.DB, repo notificationsDomain.Repository, prefsRepo notificationsDomain.PreferencesRepository, outbox EventEnqueuer) *DeliverNotificationUseCase {
	return &DeliverNotificationUseCase{db: db, repo: repo, prefsRepo: prefsRepo, outbox: outbox}
}

// Execute returns the stored notification, or nil when the tenant turned
// the bell off for this category.
func (uc *DeliverNotificationUseCase) Execute(ctx context.Context, in CreateNotificationInput) (*notificationsDomain.Notification, error) {
	var stored *notificationsDomain.Notification
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		prefs, err := loadPreferences(ctx, uc.prefsRepo)
		if err != nil {
			return err
		}
		channels := prefs.ForType(in.Type)
		now := time.Now()
		tenantID := tenant.IDFromContext(ctx)

		if channels.InApp {
			n := notificationsDomain.Notification{
				TenantID: tenantID, Type: in.Type, Title: in.Title, Message: in.Message, Link: in.Link, CreatedAt: now,
			}
			if err := uc.repo.Create(ctx, &n); err != nil {
				return err
			}
			stored = &n
		}
		if channels.Email && len(prefs.EmailRecipients) > 0 {
			return enqueue(ctx, uc.outbox, events.NotificationEmailRequested{
				Base:       events.Base{EventID: idgen.NewV4(), OccurredAt: now.UTC()},
				TenantID:   tenantID,
				Type:       string(in.Type),
				Recipients: prefs.EmailRecipients,
				Subject:    in.Title,
				Body:       in.Message,
				Link:       in.Link,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("deliverNotification: %w", err)
	}
	return stored, nil
}
