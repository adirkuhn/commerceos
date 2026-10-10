package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/locale"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// CreateNotificationInput: translated types (notificationsDomain.Types)
// need only Type + Params -- Title/Message are derived (English) for
// storage; any other type stores Title/Message as given.
type CreateNotificationInput struct {
	Type    notificationsDomain.Type
	Params  map[string]any
	Title   string
	Message string
	Link    string
}

// storedText is what a notification stores as its fallback text: the
// English translation when the type has one, else the given text.
func (in CreateNotificationInput) storedText() (params map[string]any, title, message string) {
	params = in.Params
	if params == nil {
		params = map[string]any{}
	}
	if text, ok := in.Type.Render(locale.English, params); ok {
		return params, text.Title, text.Message
	}
	return params, in.Title, in.Message
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
		params, title, message := in.storedText()
		n = notificationsDomain.Notification{
			TenantID:  tenant.IDFromContext(ctx),
			Type:      in.Type,
			Params:    params,
			Title:     title,
			Message:   message,
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
