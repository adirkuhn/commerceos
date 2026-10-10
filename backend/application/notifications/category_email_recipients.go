package notifications

import (
	"context"
	"fmt"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// CategoryEmailRecipientsUseCase answers "who gets this category by email"
// for other modules (e.g. the Message Center's staff recipients): the
// tenant's notification email recipients when the category's email toggle
// is on, else none.
type CategoryEmailRecipientsUseCase struct {
	db        *gorm.DB
	prefsRepo notificationsDomain.PreferencesRepository
}

func NewCategoryEmailRecipientsUseCase(db *gorm.DB, prefsRepo notificationsDomain.PreferencesRepository) *CategoryEmailRecipientsUseCase {
	return &CategoryEmailRecipientsUseCase{db: db, prefsRepo: prefsRepo}
}

func (uc *CategoryEmailRecipientsUseCase) Execute(ctx context.Context, c notificationsDomain.Category) ([]string, error) {
	var out []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		prefs, err := loadPreferences(ctx, uc.prefsRepo)
		if err != nil {
			return err
		}
		out = prefs.EmailRecipientsFor(c)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("categoryEmailRecipients: %w", err)
	}
	return out, nil
}
