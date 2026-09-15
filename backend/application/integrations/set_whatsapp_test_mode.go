package integrations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SetWhatsAppTestModeUseCase toggles a WhatsApp connection's test mode --
// see domain/integrations.Connection.WhatsAppTestModeEnabled's doc comment.
// Disabling clears the stored test number too, so a later re-enable can't
// silently resurrect a stale one.
type SetWhatsAppTestModeUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewSetWhatsAppTestModeUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *SetWhatsAppTestModeUseCase {
	return &SetWhatsAppTestModeUseCase{db: db, connRepo: connRepo}
}

func (uc *SetWhatsAppTestModeUseCase) Execute(ctx context.Context, connectionID string, enabled bool, testPhoneE164 string) (*integrationsDomain.Connection, error) {
	testPhoneE164 = strings.TrimSpace(testPhoneE164)
	if enabled && testPhoneE164 == "" {
		return nil, integrationsDomain.ErrWhatsAppTestPhoneRequired
	}

	var result *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if existing.Provider != integrationsDomain.ProviderWhatsApp {
			return integrationsDomain.ErrNotWhatsAppConnection
		}
		existing.WhatsAppTestModeEnabled = enabled
		if enabled {
			existing.WhatsAppTestPhoneE164 = &testPhoneE164
		} else {
			existing.WhatsAppTestPhoneE164 = nil
		}
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		result = existing
		return nil
	})
	if err != nil {
		if errors.Is(err, integrationsDomain.ErrNotWhatsAppConnection) || errors.Is(err, integrationsDomain.ErrConnectionNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("setWhatsAppTestMode: %w", err)
	}
	return result, nil
}
