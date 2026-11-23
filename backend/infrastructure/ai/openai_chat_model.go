package ai

import (
	"context"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// OpenAIChatModel implements assistant.ChatModel with tool calling over
// OpenAI's Chat Completions (the document/intent adapters use the
// Responses API; tool-calling conversations map 1:1 onto Chat Completions,
// which also lets it share NVIDIA's transport).
type OpenAIChatModel struct {
	core chatCompletionsModel
}

func NewOpenAIChatModel(apiKey string) *OpenAIChatModel {
	return &OpenAIChatModel{core: chatCompletionsModel{chat: chatCompletionsClient{
		label: "openai", apiKey: apiKey, baseURL: defaultOpenAIBaseURL, model: "gpt-4o",
	}}}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (m *OpenAIChatModel) WithBaseURL(url string) *OpenAIChatModel {
	m.core.chat.baseURL = url
	return m
}

// WithModel overrides the default gpt-4o.
func (m *OpenAIChatModel) WithModel(model string) *OpenAIChatModel {
	m.core.chat.setModel(model)
	return m
}

func (m *OpenAIChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	return m.core.next(ctx, turn)
}
