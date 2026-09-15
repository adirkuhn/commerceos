package integrations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ErrAIConnectionRequiredForWhatsAppParsing guards SetWhatsAppAIParsingUseCase
// -- enabling AI-assisted parsing with no AI connection configured would
// silently do nothing (HandleInboundMessageUseCase's resolver just returns
// nil, nil), so it's refused up front instead.
var ErrAIConnectionRequiredForWhatsAppParsing = errors.New("integrations: connect an AI provider before enabling whatsapp AI parsing")

// SetWhatsAppAIParsingUseCase toggles a WhatsApp connection's AI-assisted
// message parsing -- see domain/integrations.Connection.WhatsAppAIParsingEnabled's
// doc comment.
type SetWhatsAppAIParsingUseCase struct {
	db         *gorm.DB
	connRepo   integrationsDomain.ConnectionRepository
	aiConnRepo integrationsDomain.AIConnectionRepository
}

func NewSetWhatsAppAIParsingUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, aiConnRepo integrationsDomain.AIConnectionRepository) *SetWhatsAppAIParsingUseCase {
	return &SetWhatsAppAIParsingUseCase{db: db, connRepo: connRepo, aiConnRepo: aiConnRepo}
}

func (uc *SetWhatsAppAIParsingUseCase) Execute(ctx context.Context, connectionID string, enabled bool) (*integrationsDomain.Connection, error) {
	var result *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if existing.Provider != integrationsDomain.ProviderWhatsApp {
			return integrationsDomain.ErrNotWhatsAppConnection
		}
		if enabled {
			aiConn, err := uc.aiConnRepo.Get(ctx)
			if err != nil {
				return err
			}
			if aiConn == nil {
				return ErrAIConnectionRequiredForWhatsAppParsing
			}
		}
		existing.WhatsAppAIParsingEnabled = enabled
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		result = existing
		return nil
	})
	if err != nil {
		if errors.Is(err, integrationsDomain.ErrNotWhatsAppConnection) || errors.Is(err, integrationsDomain.ErrConnectionNotFound) ||
			errors.Is(err, ErrAIConnectionRequiredForWhatsAppParsing) {
			return nil, err
		}
		return nil, fmt.Errorf("setWhatsAppAIParsing: %w", err)
	}
	return result, nil
}
