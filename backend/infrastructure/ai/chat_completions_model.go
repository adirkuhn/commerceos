package ai

import (
	"context"
	"errors"

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
		return assistant.ChatReply{}, chatError(m.chat.label, httpStatus(err), err)
	}
	return reply, nil
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
