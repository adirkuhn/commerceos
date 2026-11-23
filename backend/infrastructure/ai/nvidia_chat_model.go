package ai

import (
	"context"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// NVIDIAChatModel implements assistant.ChatModel with tool calling over
// NVIDIA's OpenAI-compatible Chat Completions endpoint.
type NVIDIAChatModel struct {
	core chatCompletionsModel
}

func NewNVIDIAChatModel(apiKey string) *NVIDIAChatModel {
	return &NVIDIAChatModel{core: chatCompletionsModel{chat: newNVIDIAChat(apiKey)}}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (m *NVIDIAChatModel) WithBaseURL(url string) *NVIDIAChatModel {
	m.core.chat.baseURL = url
	return m
}

// WithModel overrides the default meta/llama-3.3-70b-instruct.
func (m *NVIDIAChatModel) WithModel(model string) *NVIDIAChatModel {
	m.core.chat.setModel(model)
	return m
}

// WithFallbackModels sets the models retried on a rate limit -- call after
// WithModel so the primary model is excluded from the list.
func (m *NVIDIAChatModel) WithFallbackModels(commaSeparated string) *NVIDIAChatModel {
	m.core.chat.setFallbackModels(commaSeparated)
	return m
}

func (m *NVIDIAChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	return m.core.next(ctx, turn)
}
