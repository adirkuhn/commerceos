package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// OllamaChatModel implements assistant.ChatModel with tool calling on the
// tenant's self-hosted Ollama server -- the model must support tools
// (e.g. llama3.2, qwen3); Ollama names no call ids, so they are generated.
type OllamaChatModel struct {
	baseURL string
	model   string
}

// NewOllamaChatModel requires baseURL, the tenant's Ollama server.
func NewOllamaChatModel(baseURL string) *OllamaChatModel {
	return &OllamaChatModel{baseURL: baseURL, model: defaultOllamaModel}
}

// WithModel overrides the default model.
func (m *OllamaChatModel) WithModel(model string) *OllamaChatModel {
	if model != "" {
		m.model = model
	}
	return m
}

func (m *OllamaChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	base, err := url.Parse(m.baseURL)
	if err != nil {
		return assistant.ChatReply{}, fmt.Errorf("ollamaChatModel: invalid base URL: %w", err)
	}
	tools, err := toOllamaTools(turn.Tools)
	if err != nil {
		return assistant.ChatReply{}, fmt.Errorf("ollamaChatModel: %w", err)
	}
	stream := false
	req := &api.ChatRequest{
		Model: m.model, Messages: toOllamaMessages(turn), Tools: tools, Stream: &stream,
		Options: map[string]any{"temperature": 0, "num_predict": assistantChatMaxTokens},
	}

	var reply assistant.ChatReply
	err = api.NewClient(base, http.DefaultClient).Chat(ctx, req, func(resp api.ChatResponse) error {
		reply.Content += resp.Message.Content
		for _, c := range resp.Message.ToolCalls {
			args, err := json.Marshal(c.Function.Arguments)
			if err != nil {
				return err
			}
			reply.ToolCalls = append(reply.ToolCalls, assistant.ToolCall{
				ID: generatedCallID(len(reply.ToolCalls), c.Function.Name), Name: c.Function.Name, Arguments: string(args),
			})
		}
		return nil
	})
	if err != nil {
		return assistant.ChatReply{}, chatError("ollama", m.model, ollamaStatus(err), err)
	}
	return reply, nil
}

func ollamaStatus(err error) int {
	var statusErr api.StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode
	}
	var statusPtr *api.StatusError
	if errors.As(err, &statusPtr) {
		return statusPtr.StatusCode
	}
	return 0
}

func toOllamaMessages(turn assistant.ChatTurn) []api.Message {
	out := make([]api.Message, 0, len(turn.Messages)+1)
	if turn.System != "" {
		out = append(out, api.Message{Role: "system", Content: turn.System})
	}
	for _, m := range turn.Messages {
		switch m.Role {
		case assistant.RoleUser:
			out = append(out, api.Message{Role: "user", Content: m.Content})
		case assistant.RoleTool:
			out = append(out, api.Message{Role: "tool", Content: m.Content, ToolName: m.ToolName})
		case assistant.RoleAssistant:
			msg := api.Message{Role: "assistant", Content: assistantText(m)}
			for _, c := range m.ToolCalls {
				args := api.NewToolCallFunctionArguments()
				for k, v := range argsMap(c.Arguments) {
					args.Set(k, v)
				}
				msg.ToolCalls = append(msg.ToolCalls, api.ToolCall{Function: api.ToolCallFunction{Name: c.Name, Arguments: args}})
			}
			out = append(out, msg)
		}
	}
	return out
}

// toOllamaTools converts each tool's JSON schema into Ollama's typed
// parameters through JSON, the shape both share.
func toOllamaTools(defs []assistant.ToolDefinition) (api.Tools, error) {
	out := make(api.Tools, len(defs))
	for i, d := range defs {
		raw, err := json.Marshal(d.Parameters)
		if err != nil {
			return nil, err
		}
		var params api.ToolFunctionParameters
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, fmt.Errorf("tool %s parameters: %w", d.Name, err)
		}
		out[i] = api.Tool{Type: "function", Function: api.ToolFunction{Name: d.Name, Description: d.Description, Parameters: params}}
	}
	return out, nil
}
