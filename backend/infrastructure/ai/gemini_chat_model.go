package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// GeminiChatModel implements assistant.ChatModel with Gemini function
// calling. Gemini names no call ids, so they are generated; function
// results answer by function name.
type GeminiChatModel struct {
	apiKey         string
	model          string
	baseURL        string
	fallbackModels []string
}

func NewGeminiChatModel(apiKey string) *GeminiChatModel {
	return &GeminiChatModel{apiKey: apiKey, model: defaultGeminiModel}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (m *GeminiChatModel) WithBaseURL(url string) *GeminiChatModel {
	m.baseURL = url
	return m
}

// WithModel overrides the default Gemini model.
func (m *GeminiChatModel) WithModel(model string) *GeminiChatModel {
	if model != "" {
		m.model = model
	}
	return m
}

// WithFallbackModels sets the models retried on a rate limit (the free
// tier caps some models at a few requests a day) -- call after WithModel.
func (m *GeminiChatModel) WithFallbackModels(commaSeparated string) *GeminiChatModel {
	m.fallbackModels = nil
	for _, f := range strings.Split(commaSeparated, ",") {
		if f = strings.TrimSpace(f); f != "" && f != m.model {
			m.fallbackModels = append(m.fallbackModels, f)
		}
	}
	return m
}

func (m *GeminiChatModel) Next(ctx context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	cfg := &genai.ClientConfig{APIKey: m.apiKey, Backend: genai.BackendGeminiAPI}
	if m.baseURL != "" {
		cfg.HTTPOptions = genai.HTTPOptions{BaseURL: m.baseURL}
	}
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return assistant.ChatReply{}, fmt.Errorf("geminiChatModel: build client: %w", err)
	}

	temperature := float32(0)
	config := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: assistantChatMaxTokens * 4, // thinking models spend part of it reasoning
		Tools:           toGeminiTools(turn.Tools),
	}
	if turn.System != "" {
		config.SystemInstruction = genai.NewContentFromText(turn.System, genai.RoleUser)
	}
	contents := toGeminiContents(turn.Messages)

	models := append([]string{m.model}, m.fallbackModels...)
	var lastErr error
	for i, model := range models {
		resp, err := client.Models.GenerateContent(ctx, model, contents, config)
		if err == nil {
			return geminiReply(resp), nil
		}
		lastErr = err
		var apiErr genai.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != 429 {
			break
		}
		if i < len(models)-1 {
			log.Printf("geminiChatModel: %s rate-limited, falling back to %s", model, models[i+1])
		}
	}
	var apiErr genai.APIError
	status := 0
	if errors.As(lastErr, &apiErr) {
		status = apiErr.Code
	}
	return assistant.ChatReply{}, chatError("gemini", status, lastErr)
}

func geminiReply(resp *genai.GenerateContentResponse) assistant.ChatReply {
	var reply assistant.ChatReply
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return reply
	}
	var text []string
	for _, p := range resp.Candidates[0].Content.Parts {
		switch {
		case p.FunctionCall != nil:
			args, _ := json.Marshal(p.FunctionCall.Args)
			if p.FunctionCall.Args == nil {
				args = []byte("{}")
			}
			id := p.FunctionCall.ID
			if id == "" {
				id = generatedCallID(len(reply.ToolCalls), p.FunctionCall.Name)
			}
			reply.ToolCalls = append(reply.ToolCalls, assistant.ToolCall{ID: id, Name: p.FunctionCall.Name, Arguments: string(args)})
		case p.Text != "" && !p.Thought:
			text = append(text, p.Text)
		}
	}
	reply.Content = strings.Join(text, "\n")
	return reply
}

func toGeminiContents(msgs []assistant.Message) []*genai.Content {
	var out []*genai.Content
	add := func(role genai.Role, parts ...*genai.Part) {
		if len(parts) == 0 {
			return
		}
		if n := len(out); n > 0 && out[n-1].Role == string(role) {
			out[n-1].Parts = append(out[n-1].Parts, parts...)
			return
		}
		out = append(out, &genai.Content{Role: string(role), Parts: parts})
	}
	for _, m := range msgs {
		switch m.Role {
		case assistant.RoleUser:
			add(genai.RoleUser, genai.NewPartFromText(m.Content))
		case assistant.RoleTool:
			// Matched to its call by name: call ids may be generated ones
			// Gemini never saw.
			add(genai.RoleUser, &genai.Part{FunctionResponse: &genai.FunctionResponse{Name: m.ToolName, Response: toolResponse(m.Content)}})
		case assistant.RoleAssistant:
			var parts []*genai.Part
			if text := assistantText(m); text != "" {
				parts = append(parts, genai.NewPartFromText(text))
			}
			for _, c := range m.ToolCalls {
				parts = append(parts, &genai.Part{FunctionCall: &genai.FunctionCall{Name: c.Name, Args: argsMap(c.Arguments)}})
			}
			add(genai.RoleModel, parts...)
		}
	}
	return out
}

// toolResponse: Gemini takes a function's result as a JSON object; any
// other result is wrapped as {"output": ...}.
func toolResponse(content string) map[string]any {
	var obj map[string]any
	if err := json.Unmarshal([]byte(content), &obj); err == nil && obj != nil {
		return obj
	}
	return map[string]any{"output": content}
}

func toGeminiTools(defs []assistant.ToolDefinition) []*genai.Tool {
	if len(defs) == 0 {
		return nil
	}
	decls := make([]*genai.FunctionDeclaration, len(defs))
	for i, d := range defs {
		decls[i] = &genai.FunctionDeclaration{Name: d.Name, Description: d.Description, ParametersJsonSchema: d.Parameters}
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}
}
