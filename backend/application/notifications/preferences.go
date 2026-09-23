package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// EventEnqueuer is the one outbox operation these use cases need
// (satisfied by outbox.Repository).
type EventEnqueuer interface {
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

func enqueue(ctx context.Context, outbox EventEnqueuer, e events.DomainEvent) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", e.EventName(), err)
	}
	return outbox.Enqueue(ctx, e.EventName(), e.GetEventID(), payload, e.GetOccurredAt())
}

// loadPreferences returns the tenant's saved preferences or the defaults.
// Must run inside tenant.WithTenant.
func loadPreferences(ctx context.Context, repo notificationsDomain.PreferencesRepository) (*notificationsDomain.Preferences, error) {
	p, err := repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = notificationsDomain.DefaultPreferences(tenant.IDFromContext(ctx))
	}
	return p, nil
}

type GetPreferencesUseCase struct {
	db   *gorm.DB
	repo notificationsDomain.PreferencesRepository
}

func NewGetPreferencesUseCase(db *gorm.DB, repo notificationsDomain.PreferencesRepository) *GetPreferencesUseCase {
	return &GetPreferencesUseCase{db: db, repo: repo}
}

func (uc *GetPreferencesUseCase) Execute(ctx context.Context) (*notificationsDomain.Preferences, error) {
	var result *notificationsDomain.Preferences
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := loadPreferences(ctx, uc.repo)
		result = p
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getNotificationPreferences: %w", err)
	}
	return result, nil
}

type UpdatePreferencesUseCase struct {
	db     *gorm.DB
	repo   notificationsDomain.PreferencesRepository
	outbox EventEnqueuer
}

func NewUpdatePreferencesUseCase(db *gorm.DB, repo notificationsDomain.PreferencesRepository, outbox EventEnqueuer) *UpdatePreferencesUseCase {
	return &UpdatePreferencesUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *UpdatePreferencesUseCase) Execute(ctx context.Context, in notificationsDomain.PreferencesInput) (*notificationsDomain.Preferences, error) {
	var result *notificationsDomain.Preferences
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now().UTC()
		tenantID := tenant.IDFromContext(ctx)
		p, err := notificationsDomain.NewPreferences(tenantID, in, now)
		if err != nil {
			return err
		}
		if err := uc.repo.Upsert(ctx, p); err != nil {
			return err
		}
		result = p
		return enqueue(ctx, uc.outbox, events.NotificationPreferencesUpdated{
			Base: events.Base{EventID: idgen.NewV4(), OccurredAt: now}, TenantID: tenantID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("updateNotificationPreferences: %w", err)
	}
	return result, nil
}
