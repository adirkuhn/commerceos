package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// AssistantModelResolver implements assistant.ChatModelResolver from the
// tenant's AI connection: every provider has a tool-calling adapter.
type AssistantModelResolver struct {
	db               *gorm.DB
	aiConnRepo       integrationsDomain.AIConnectionRepository
	encryptionSecret string
}

func NewAssistantModelResolver(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, encryptionSecret string) *AssistantModelResolver {
	return &AssistantModelResolver{db: db, aiConnRepo: aiConnRepo, encryptionSecret: encryptionSecret}
}

func (r *AssistantModelResolver) Resolve(ctx context.Context) (assistant.ChatModel, error) {
	var conn *integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		var err error
		conn, err = r.aiConnRepo.Get(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("assistantModelResolver: %w", err)
	}
	if conn == nil {
		return nil, nil
	}

	apiKey, err := r.apiKey(conn)
	if err != nil {
		return nil, err
	}
	return buildChatModel(conn, apiKey)
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

func (r *AssistantModelResolver) apiKey(conn *integrationsDomain.AIConnection) (string, error) {
	if conn.APIKeyEncrypted == "" {
		return "", nil
	}
	key, err := crypto.Decrypt(r.encryptionSecret, conn.APIKeyEncrypted)
	if err != nil {
		return "", fmt.Errorf("assistantModelResolver: decrypt key: %w", err)
	}
	return key, nil
}
