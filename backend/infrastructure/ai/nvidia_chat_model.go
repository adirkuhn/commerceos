package ai

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// nvidiaChatMaxTokens bounds one assistant step -- a final answer or a
// handful of tool calls, never a long document.
const nvidiaChatMaxTokens = 2048

// tooManyStepsText is how an app-written NoticeTooManySteps message reads
// to the model on the next turn (the user sees it translated instead).
const tooManyStepsText = "I needed too many lookups without reaching an answer, so I stopped and asked the user to add detail."

// NVIDIAChatModel implements assistant.ChatModel with tool calling over
// NVIDIA's OpenAI-compatible Chat Completions endpoint.
type NVIDIAChatModel struct {
	chat nvidiaChat
}

func NewNVIDIAChatModel(apiKey string) *NVIDIAChatModel {
	return &NVIDIAChatModel{chat: newNVIDIAChat(apiKey)}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (m *NVIDIAChatModel) WithBaseURL(url string) *NVIDIAChatModel {
	m.chat.baseURL = url
	return m
}

// WithModel overrides the default meta/llama-3.3-70b-instruct.
func (m *NVIDIAChatModel) WithModel(model string) *NVIDIAChatModel {
	m.chat.setModel(model)
	return m
}

// WithFallbackModels sets the models retried on a rate limit -- call after
// WithModel so the primary model is excluded from the list.
func (m *NVIDIAChatModel) WithFallbackModels(commaSeparated string) *NVIDIAChatModel {
	m.chat.setFallbackModels(commaSeparated)
	return m
}

func (m *NVIDIAChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	params := openai.ChatCompletionNewParams{
		Messages:    toChatMessages(turn),
		Tools:       toChatTools(turn.Tools),
		Temperature: openai.Float(0),
		MaxTokens:   openai.Int(nvidiaChatMaxTokens),
	}
	client := m.chat.newClient()
	var reply assistant.ChatReply
	err := m.chat.withFallback(func(model string) error {
		params.Model = model
		resp, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			return errors.New("empty response")
		}
		msg := resp.Choices[0].Message
		reply = assistant.ChatReply{Content: msg.Content}
		for _, c := range msg.ToolCalls {
			reply.ToolCalls = append(reply.ToolCalls, assistant.ToolCall{ID: c.ID, Name: c.Function.Name, Arguments: c.Function.Arguments})
		}
		return nil
	})
	if err != nil {
		if isTransient(err) {
			return assistant.ChatReply{}, fmt.Errorf("nvidiaChatModel: %w: %w", assistant.ErrModelBusy, err)
		}
		return assistant.ChatReply{}, fmt.Errorf("nvidiaChatModel: %w", err)
	}
	return reply, nil
}

// isTransient: rate limit, quota or overload on NVIDIA's side, or a
// network timeout -- worth waiting in the queue for.
func isTransient(err error) bool {
	switch httpStatus(err) {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func toChatMessages(turn assistant.ChatTurn) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(turn.Messages)+1)
	if turn.System != "" {
		out = append(out, openai.SystemMessage(turn.System))
	}
	for _, msg := range turn.Messages {
		switch msg.Role {
		case assistant.RoleUser:
			out = append(out, openai.UserMessage(msg.Content))
		case assistant.RoleTool:
			out = append(out, openai.ToolMessage(msg.Content, msg.ToolCallID))
		case assistant.RoleAssistant:
			out = append(out, toAssistantMessage(msg))
		}
	}
	return out
}

func toAssistantMessage(msg assistant.Message) openai.ChatCompletionMessageParamUnion {
	content := msg.Content
	if msg.Notice == assistant.NoticeTooManySteps {
		content = tooManyStepsText
	}
	if len(msg.ToolCalls) == 0 {
		return openai.AssistantMessage(content)
	}
	p := openai.ChatCompletionAssistantMessageParam{}
	if content != "" {
		p.Content.OfString = openai.String(content)
	}
	for _, c := range msg.ToolCalls {
		p.ToolCalls = append(p.ToolCalls, openai.ChatCompletionMessageToolCallParam{
			ID:       c.ID,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{Name: c.Name, Arguments: c.Arguments},
		})
	}
	return openai.ChatCompletionMessageParamUnion{OfAssistant: &p}
}

func toChatTools(defs []assistant.ToolDefinition) []openai.ChatCompletionToolParam {
	out := make([]openai.ChatCompletionToolParam, len(defs))
	for i, d := range defs {
		out[i] = openai.ChatCompletionToolParam{Function: shared.FunctionDefinitionParam{
			Name: d.Name, Description: openai.String(d.Description), Parameters: shared.FunctionParameters(d.Parameters),
		}}
	}
	return out
}
