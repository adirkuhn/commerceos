package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ParserResolver implements procurement.AIParserResolver -- looks up the
// tenant's AIConnection and constructs the matching per-vendor adapter.
// This is the one place that knows about every adapter; everything
// downstream only ever depends on the AIDocumentParser port.
type ParserResolver struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
}

func NewParserResolver(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string) *ParserResolver {
	return &ParserResolver{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret}
}

func (r *ParserResolver) Resolve(ctx context.Context) (procurementDomain.AIDocumentParser, error) {
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
		return nil, fmt.Errorf("aiParserResolver: %w", err)
	}
	if conn == nil {
		return nil, nil
	}

	var apiKey string
	if conn.APIKeyEncrypted != "" {
		decrypted, err := crypto.Decrypt(r.encryptionSecret, conn.APIKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("aiParserResolver: decrypt key: %w", err)
		}
		apiKey = decrypted
	}

	switch conn.Provider {
	case integrationsDomain.AIProviderOpenAI:
		p := NewOpenAIDocumentParser(apiKey).WithModel(conn.Model)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderAnthropic:
		p := NewAnthropicDocumentParser(apiKey).WithModel(conn.Model)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderGemini:
		p := NewGeminiDocumentParser(apiKey).WithModel(conn.Model)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderNVIDIA:
		p := NewNVIDIADocumentParser(apiKey).WithModel(conn.Model).WithFallbackModels(conn.FallbackModels)
		if conn.BaseURL != "" {
			p = p.WithBaseURL(conn.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderOllama:
		return NewOllamaDocumentParser(conn.BaseURL).WithModel(conn.Model), nil
	default:
		return nil, fmt.Errorf("aiParserResolver: %w", integrationsDomain.ErrProviderNotSupported)
	}
}
