package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// WhatsAppIntentParserResolver implements whatsapp.AIIntentParserResolver
// over the tenant's ordered AI connections (see router.go).
type WhatsAppIntentParserResolver struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
}

func NewWhatsAppIntentParserResolver(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string) *WhatsAppIntentParserResolver {
	return &WhatsAppIntentParserResolver{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret}
}

func (r *WhatsAppIntentParserResolver) Resolve(ctx context.Context) (whatsappDomain.AIIntentParser, error) {
	conns, err := loadConnections(ctx, r.db, r.aiConnRepo, r.encryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("whatsAppIntentParserResolver: %w", err)
	}
	targets := buildTargets(conns, buildIntentParser)
	switch {
	case len(targets) == 0 && len(conns) > 0:
		return nil, fmt.Errorf("whatsAppIntentParserResolver: %w", integrationsDomain.ErrProviderNotSupported)
	case len(targets) == 0:
		return nil, nil
	case len(targets) == 1:
		return targets[0].target, nil
	}
	return routedIntentParser{targets: targets}, nil
}

// routedIntentParser tries each connection's parser in order.
type routedIntentParser struct {
	targets []routeTarget[whatsappDomain.AIIntentParser]
}

func (p routedIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	return route(ctx, p.targets, func(t whatsappDomain.AIIntentParser) (whatsappDomain.ParsedIntent, error) {
		return t.ParseIntent(ctx, text)
	})
}

func buildIntentParser(lc loadedConnection) (whatsappDomain.AIIntentParser, error) {
	c, key := lc.conn, lc.apiKey
	switch c.Provider {
	case integrationsDomain.AIProviderOpenAI:
		p := NewOpenAIWhatsAppIntentParser(key).WithModel(c.Model)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderAnthropic:
		p := NewAnthropicWhatsAppIntentParser(key).WithModel(c.Model)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderGemini:
		p := NewGeminiWhatsAppIntentParser(key).WithModel(c.Model).WithFallbackModels(c.FallbackModels)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderOllama:
		return NewOllamaWhatsAppIntentParser(c.BaseURL).WithModel(c.Model), nil
	case integrationsDomain.AIProviderNVIDIA:
		p := NewNVIDIAWhatsAppIntentParser(key).WithModel(c.Model).WithFallbackModels(c.FallbackModels)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	default:
		return nil, integrationsDomain.ErrProviderNotSupported
	}
}
