package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// ParserResolver implements procurement.AIParserResolver over the tenant's
// ordered AI connections (see router.go). Everything downstream only ever
// depends on the AIDocumentParser port.
type ParserResolver struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
}

func NewParserResolver(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string) *ParserResolver {
	return &ParserResolver{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret}
}

func (r *ParserResolver) Resolve(ctx context.Context) (procurementDomain.AIDocumentParser, error) {
	conns, err := loadConnections(ctx, r.db, r.aiConnRepo, r.encryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("aiParserResolver: %w", err)
	}
	targets := buildTargets(conns, buildDocumentParser)
	switch {
	case len(targets) == 0 && len(conns) > 0:
		return nil, fmt.Errorf("aiParserResolver: %w", integrationsDomain.ErrProviderNotSupported)
	case len(targets) == 0:
		return nil, nil
	case len(targets) == 1:
		return targets[0].target, nil
	}
	return routedDocumentParser{targets: targets}, nil
}

// routedDocumentParser tries each connection's parser in order.
type routedDocumentParser struct {
	targets []routeTarget[procurementDomain.AIDocumentParser]
}

func (p routedDocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	return route(ctx, p.targets, func(t procurementDomain.AIDocumentParser) (procurementDomain.DocumentExtractionResult, error) {
		return t.ExtractLineItems(ctx, doc)
	})
}

func buildDocumentParser(lc loadedConnection) (procurementDomain.AIDocumentParser, error) {
	c, key := lc.conn, lc.apiKey
	switch c.Provider {
	case integrationsDomain.AIProviderOpenAI:
		p := NewOpenAIDocumentParser(key).WithModel(c.Model)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderAnthropic:
		p := NewAnthropicDocumentParser(key).WithModel(c.Model)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderGemini:
		p := NewGeminiDocumentParser(key).WithModel(c.Model)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	case integrationsDomain.AIProviderOllama:
		return NewOllamaDocumentParser(c.BaseURL).WithModel(c.Model), nil
	case integrationsDomain.AIProviderNVIDIA:
		p := NewNVIDIADocumentParser(key).WithModel(c.Model).WithFallbackModels(c.FallbackModels)
		if c.BaseURL != "" {
			p = p.WithBaseURL(c.BaseURL)
		}
		return p, nil
	default:
		return nil, integrationsDomain.ErrProviderNotSupported
	}
}
