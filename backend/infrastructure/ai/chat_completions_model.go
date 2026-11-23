package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// chatCompletionsModel implements assistant.ChatModel with tool calling
// over OpenAI-compatible Chat Completions (NVIDIA, OpenAI).
type chatCompletionsModel struct {
	chat chatCompletionsClient
}

func (m chatCompletionsModel) next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	params := openai.ChatCompletionNewParams{
		Messages:    toChatMessages(turn),
		Tools:       toChatTools(turn.Tools),
		Temperature: openai.Float(0),
		MaxTokens:   openai.Int(assistantChatMaxTokens),
	}
	client := m.chat.newClient()
	var reply assistant.ChatReply
	err := m.chat.withFallback(func(model string) error {
		params.Model = model
		resp, err := client.Chat.Completions.New(ctx, params, m.chat.requestOptions(model)...)
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
		return assistant.ChatReply{}, chatError(m.chat.label, m.chat.model, httpStatus(err), withProviderReason(err))
	}
	return reply, nil
}

// withProviderReason adds the provider's response body to err when the SDK
// left it out of the message -- NVIDIA answers some failures (a retired
// model's 410) with a body the OpenAI SDK doesn't parse as an error.
func withProviderReason(err error) error {
	var apiErr *openai.Error
	if !errors.As(err, &apiErr) {
		return err
	}
	raw := strings.TrimSpace(apiErr.RawJSON())
	if raw == "" && apiErr.Response != nil {
		raw = strings.TrimSpace(string(responseBody(apiErr.DumpResponse(true))))
	}
	if raw == "" || strings.Contains(err.Error(), raw) {
		return err
	}
	const maxReason = 300
	if len(raw) > maxReason {
		raw = raw[:maxReason] + "…"
	}
	return fmt.Errorf("%w: %s", err, raw)
}

// responseBody is the body part of an HTTP response dump.
func responseBody(dump []byte) []byte {
	if i := bytes.Index(dump, []byte("\r\n\r\n")); i >= 0 {
		return dump[i+4:]
	}
	return nil
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
	content := assistantText(msg)
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
