package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// AnthropicChatModel implements assistant.ChatModel with Claude's Messages
// API tool use. Claude wants strictly alternating user/assistant turns, so
// tool results ride in a user turn and neighbouring same-role messages are
// merged into one turn.
type AnthropicChatModel struct {
	apiKey  string
	model   anthropic.Model
	baseURL string
}

func NewAnthropicChatModel(apiKey string) *AnthropicChatModel {
	return &AnthropicChatModel{apiKey: apiKey, model: anthropic.ModelClaudeSonnet5}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (m *AnthropicChatModel) WithBaseURL(url string) *AnthropicChatModel {
	m.baseURL = url
	return m
}

// WithModel overrides the default Claude Sonnet.
func (m *AnthropicChatModel) WithModel(model string) *AnthropicChatModel {
	if model != "" {
		m.model = anthropic.Model(model)
	}
	return m
}

func (m *AnthropicChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	opts := []option.RequestOption{option.WithAPIKey(m.apiKey), option.WithMaxRetries(0)}
	if m.baseURL != "" {
		opts = append(opts, option.WithBaseURL(m.baseURL))
	}
	client := anthropic.NewClient(opts...)

	params := anthropic.MessageNewParams{
		Model:       m.model,
		MaxTokens:   assistantChatMaxTokens,
		Temperature: anthropic.Float(0),
		Messages:    toAnthropicMessages(turn.Messages),
		Tools:       toAnthropicTools(turn.Tools),
	}
	if turn.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: turn.System}}
	}
	msg, err := client.Messages.New(ctx, params)
	if err != nil {
		var apiErr *anthropic.Error
		status := 0
		if errors.As(err, &apiErr) {
			status = apiErr.StatusCode
		}
		return assistant.ChatReply{}, chatError("anthropic", status, err)
	}

	var reply assistant.ChatReply
	var text []string
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			text = append(text, block.Text)
		case "tool_use":
			reply.ToolCalls = append(reply.ToolCalls, assistant.ToolCall{ID: block.ID, Name: block.Name, Arguments: string(block.Input)})
		}
	}
	reply.Content = strings.Join(text, "\n")
	return reply, nil
}

func toAnthropicMessages(msgs []assistant.Message) []anthropic.MessageParam {
	var out []anthropic.MessageParam
	add := func(role anthropic.MessageParamRole, blocks ...anthropic.ContentBlockParamUnion) {
		if len(blocks) == 0 {
			return
		}
		if n := len(out); n > 0 && out[n-1].Role == role {
			out[n-1].Content = append(out[n-1].Content, blocks...)
			return
		}
		out = append(out, anthropic.MessageParam{Role: role, Content: blocks})
	}
	for _, m := range msgs {
		switch m.Role {
		case assistant.RoleUser:
			add(anthropic.MessageParamRoleUser, anthropic.NewTextBlock(m.Content))
		case assistant.RoleTool:
			add(anthropic.MessageParamRoleUser, anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false))
		case assistant.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			if text := assistantText(m); text != "" {
				blocks = append(blocks, anthropic.NewTextBlock(text))
			}
			for _, c := range m.ToolCalls {
				blocks = append(blocks, anthropic.NewToolUseBlock(c.ID, argsMap(c.Arguments), c.Name))
			}
			add(anthropic.MessageParamRoleAssistant, blocks...)
		}
	}
	return out
}

func toAnthropicTools(defs []assistant.ToolDefinition) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, len(defs))
	for i, d := range defs {
		schema := anthropic.ToolInputSchemaParam{Properties: d.Parameters["properties"]}
		if req, ok := d.Parameters["required"].([]string); ok {
			schema.Required = req
		}
		out[i] = anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name: d.Name, Description: anthropic.String(d.Description), InputSchema: schema,
		}}
	}
	return out
}
