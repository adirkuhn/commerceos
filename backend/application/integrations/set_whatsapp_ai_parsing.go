package integrations

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ErrAIConnectionRequiredForWhatsAppParsing guards SetWhatsAppAIParsingUseCase
// -- enabling AI-assisted parsing with no AI connection configured would
// silently do nothing (HandleInboundMessageUseCase's resolver just returns
// nil, nil), so it's refused up front instead.
var ErrAIConnectionRequiredForWhatsAppParsing = errs.New("WHATSAPP_AI_PARSING_NEEDS_AI_CONNECTION", "integrations: connect an AI provider before enabling whatsapp AI parsing")

// SetWhatsAppAIParsingUseCase toggles a WhatsApp connection's AI-assisted
// message parsing -- see domain/integrations.Connection.WhatsAppAIParsingEnabled's
// doc comment.
type SetWhatsAppAIParsingUseCase struct {
	db         *gorm.DB
	connRepo   integrationsDomain.ConnectionRepository
	aiConnRepo integrationsDomain.AIConnectionRepository
	outbox     eventrecord.Enqueuer
}

func NewSetWhatsAppAIParsingUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, aiConnRepo integrationsDomain.AIConnectionRepository, outbox eventrecord.Enqueuer) *SetWhatsAppAIParsingUseCase {
	return &SetWhatsAppAIParsingUseCase{db: db, connRepo: connRepo, aiConnRepo: aiConnRepo, outbox: outbox}
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
			conns, err := uc.aiConnRepo.List(ctx)
			if err != nil {
				return err
			}
			if !anyEnabled(conns) {
				return ErrAIConnectionRequiredForWhatsAppParsing
			}
		}
		before := *existing
		existing.WhatsAppAIParsingEnabled = enabled
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		result = existing
		return recordConnectionUpdated(ctx, uc.outbox, *existing, eventrecord.NewChanges().
			Set("whatsapp_ai_parsing_enabled", before.WhatsAppAIParsingEnabled, existing.WhatsAppAIParsingEnabled))
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

func anyEnabled(conns []integrationsDomain.AIConnection) bool {
	for _, c := range conns {
		if c.Enabled {
			return true
		}
	}
	return false
}
