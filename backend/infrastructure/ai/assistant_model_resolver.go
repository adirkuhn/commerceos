package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// AssistantModelResolver implements assistant.ChatModelResolver over the
// tenant's ordered AI connections (see router.go): every provider has a
// tool-calling adapter.
type AssistantModelResolver struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
}

func NewAssistantModelResolver(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string) *AssistantModelResolver {
	return &AssistantModelResolver{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret}
}

func (r *AssistantModelResolver) Resolve(ctx context.Context) (assistant.ChatModel, error) {
	conns, err := loadConnections(ctx, r.db, r.aiConnRepo, r.encryptionSecret)
	if err != nil {
		return nil, fmt.Errorf("assistantModelResolver: %w", err)
	}
	targets := buildTargets(conns, func(lc loadedConnection) (assistant.ChatModel, error) {
		return buildChatModel(&lc.conn, lc.apiKey)
	})
	switch {
	case len(targets) == 0 && len(conns) > 0:
		return nil, fmt.Errorf("assistantModelResolver: %w", assistant.ErrProviderUnsupported)
	case len(targets) == 0:
		return nil, nil
	case len(targets) == 1:
		return targets[0].target, nil
	}
	return routedChatModel{targets: targets}, nil
}

// routedChatModel tries each connection's model in order. When all fail
// and any was busy, the joined error still wraps ErrModelBusy, so the
// request waits in the queue.
type routedChatModel struct {
	targets []routeTarget[assistant.ChatModel]
}

func (m routedChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	return route(ctx, m.targets, func(t assistant.ChatModel) (assistant.ChatReply, error) {
		return t.Next(ctx, turn)
	})
}

// buildChatModel is the one place that maps a provider to its tool-calling
// adapter; BaseURL overrides each provider's endpoint (Ollama's is its
// server).
func buildChatModel(conn *integrationsDomain.AIConnection, apiKey string) (assistant.ChatModel, error) {
	base := conn.BaseURL
	switch conn.Provider {
	case integrationsDomain.AIProviderNVIDIA:
		m := NewNVIDIAChatModel(apiKey).WithModel(conn.Model).WithFallbackModels(conn.FallbackModels)
		if base != "" {
			m = m.WithBaseURL(base)
		}
		return m, nil
	case integrationsDomain.AIProviderOpenAI:
		m := NewOpenAIChatModel(apiKey).WithModel(conn.Model)
		if base != "" {
			m = m.WithBaseURL(base)
		}
		return m, nil
	case integrationsDomain.AIProviderAnthropic:
		m := NewAnthropicChatModel(apiKey).WithModel(conn.Model)
		if base != "" {
			m = m.WithBaseURL(base)
		}
		return m, nil
	case integrationsDomain.AIProviderGemini:
		m := NewGeminiChatModel(apiKey).WithModel(conn.Model).WithFallbackModels(conn.FallbackModels)
		if base != "" {
			m = m.WithBaseURL(base)
		}
		return m, nil
	case integrationsDomain.AIProviderOllama:
		return NewOllamaChatModel(base).WithModel(conn.Model), nil
	default:
		return nil, fmt.Errorf("assistantModelResolver: %s: %w", conn.Provider, assistant.ErrProviderUnsupported)
	}
}
