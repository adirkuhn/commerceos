package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// WhatsAppIntentParserResolver implements whatsapp.AIIntentParserResolver
// -- same shape as ParserResolver (procurement's), the one place that
// knows about every adapter.
type WhatsAppIntentParserResolver struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
}

func NewWhatsAppIntentParserResolver(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string) *WhatsAppIntentParserResolver {
	return &WhatsAppIntentParserResolver{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret}
}

func (r *WhatsAppIntentParserResolver) Resolve(ctx context.Context) (whatsappDomain.AIIntentParser, error) {
	var conn *integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		c, err := r.aiConnRepo.Get(ctx)
		if err != nil {
			return err
		}
		conn = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("whatsAppIntentParserResolver: %w", err)
	}
	if conn == nil {
		return nil, nil
	}

	var apiKey string
	if conn.APIKeyEncrypted != "" {
		decrypted, err := crypto.Decrypt(r.encryptionSecret, conn.APIKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("whatsAppIntentParserResolver: decrypt key: %w", err)
		}
		apiKey = decrypted
	}

	switch conn.Provider {
	case integrationsDomain.AIProviderOpenAI:
		p := NewOpenAIWhatsAppIntentParser(apiKey).WithModel(conn.Model)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderAnthropic:
		p := NewAnthropicWhatsAppIntentParser(apiKey).WithModel(conn.Model)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderGemini:
		p := NewGeminiWhatsAppIntentParser(apiKey).WithModel(conn.Model).WithFallbackModels(conn.FallbackModels)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderNVIDIA:
		p := NewNVIDIAWhatsAppIntentParser(apiKey).WithModel(conn.Model).WithFallbackModels(conn.FallbackModels)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderOllama:
		return NewOllamaWhatsAppIntentParser(conn.BaseURL).WithModel(conn.Model), nil
	default:
		return nil, fmt.Errorf("whatsAppIntentParserResolver: %w", integrationsDomain.ErrProviderNotSupported)
	}
}
