package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const (
	// MaxSteps caps model calls per turn, so a model looping on tools
	// can't burn the free quota.
	MaxSteps = 8
	// MaxToolResultChars keeps one tool result from flooding the context.
	MaxToolResultChars = 12000
	maxErrorDetail     = 500
	askUserToolName    = "ask_user"
)

// errStopped: the request left processing (the user cancelled) while the
// worker was running -- whatever the attempt produced is dropped.
var errStopped = errors.New("assistant request no longer processing")

var askUserTool = assistant.ToolDefinition{
	Name: askUserToolName,
	Description: "Ask the user a clarifying question when the request is ambiguous or missing something you " +
		"can't look up (e.g. two customers match a name). The conversation pauses until they answer.",
	Parameters: map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"question": map[string]any{"type": "string", "description": "The question, in the user's language."}},
		"required":             []string{"question"},
		"additionalProperties": false,
	},
}

// ProcessRequestUseCase runs one attempt of a claimed (processing)
// request: model steps with tool calls until it answers, asks the user,
// proposes a change, or fails -- a failure puts the request back in the
// queue. ctx carries the requester's tenant, user and role, so every tool
// runs as them.
type ProcessRequestUseCase struct {
	db          *gorm.DB
	requests    assistant.RequestRepository
	messages    assistant.MessageRepository
	proposals   assistant.ProposalRepository
	models      assistant.ChatModelResolver
	tools       map[string]assistant.Tool
	actions     map[string]ProposalAction
	definitions []assistant.ToolDefinition
	outbox      eventrecord.Enqueuer
	now         func() time.Time
}

// ProcessDeps are ProcessRequestUseCase's ports: Tools are the read tools,
// Actions the changes the model may propose (by their Tool name).
type ProcessDeps struct {
	Requests  assistant.RequestRepository
	Messages  assistant.MessageRepository
	Proposals assistant.ProposalRepository
	Models    assistant.ChatModelResolver
	Tools     []assistant.Tool
	Actions   []ProposalAction
	Outbox    eventrecord.Enqueuer
	Now       func() time.Time
}

func NewProcessRequestUseCase(db *gorm.DB, d ProcessDeps) *ProcessRequestUseCase {
	uc := &ProcessRequestUseCase{
		db: db, requests: d.Requests, messages: d.Messages, proposals: d.Proposals, models: d.Models,
		tools: map[string]assistant.Tool{}, actions: map[string]ProposalAction{}, outbox: d.Outbox, now: d.Now,
	}
	for _, t := range d.Tools {
		def := t.Definition()
		uc.tools[def.Name] = t
		uc.definitions = append(uc.definitions, def)
	}
	for _, a := range d.Actions {
		def := a.Tool()
		uc.actions[def.Name] = a
		uc.definitions = append(uc.definitions, def)
	}
	uc.definitions = append(uc.definitions, askUserTool)
	return uc
}

func (uc *ProcessRequestUseCase) Execute(ctx context.Context, req assistant.Request) error {
	err := uc.attempt(ctx, req)
	if errors.Is(err, errStopped) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("processAssistantRequest %s: %w", req.ID, err)
	}
	return nil
}

func (uc *ProcessRequestUseCase) attempt(ctx context.Context, req assistant.Request) error {
	model, err := uc.models.Resolve(ctx)
	switch {
	case errors.Is(err, assistant.ErrProviderUnsupported):
		return uc.retryLater(ctx, req.ID, assistant.WaitProviderUnsupported, err)
	case err != nil:
		return uc.retryLater(ctx, req.ID, assistant.WaitModelError, err)
	case model == nil:
		return uc.retryLater(ctx, req.ID, assistant.WaitNoProvider, nil)
	}

	var history []assistant.Message
	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		history, err = uc.messages.ListByRequest(ctx, req.ID)
		return err
	}); err != nil {
		return err
	}

	toolCalls := 0
	for step := 0; step < MaxSteps; step++ {
		reply, err := model.Next(ctx, assistant.ChatTurn{System: systemPrompt(uc.now()), Messages: history, Tools: uc.definitions})
		if err != nil {
			reason := assistant.WaitModelError
			if errors.Is(err, assistant.ErrModelBusy) {
				reason = assistant.WaitModelBusy
			}
			return uc.retryLater(ctx, req.ID, reason, err)
		}
		if len(reply.ToolCalls) == 0 {
			return uc.finish(ctx, req.ID, assistant.Message{Role: assistant.RoleAssistant, Content: reply.Content},
				(*assistant.Request).Answer, events.AssistantRequestAnswered, eventrecord.Data{"tool_calls": toolCalls})
		}
		if question, ok := askedQuestion(reply.ToolCalls); ok {
			return uc.finish(ctx, req.ID, assistant.Message{Role: assistant.RoleAssistant, Content: question, Question: true},
				(*assistant.Request).AskUser, events.AssistantRequestQuestionAsked, eventrecord.Data{"tool_calls": toolCalls})
		}

		var step []assistant.Message
		if call, action, ok := uc.proposalCall(reply.ToolCalls); ok {
			payload, preview, err := action.Prepare(ctx, json.RawMessage(call.Arguments))
			if err == nil {
				return uc.propose(ctx, req.ID, reply.Content, call, action.Kind(), payload, preview)
			}
			step = rejectedProposalStep(req.ID, reply.Content, call, err)
		} else {
			step = uc.runTools(ctx, req.ID, reply)
		}
		if err := uc.appendWhileProcessing(ctx, req.ID, step); err != nil {
			return err
		}
		history = append(history, step...)
		toolCalls += len(reply.ToolCalls)
	}
	return uc.finish(ctx, req.ID, assistant.Message{Role: assistant.RoleAssistant, Notice: assistant.NoticeTooManySteps, Question: true},
		(*assistant.Request).AskUser, events.AssistantRequestQuestionAsked,
		eventrecord.Data{"tool_calls": toolCalls, "notice": string(assistant.NoticeTooManySteps)})
}

// askedQuestion finds an ask_user call; any other calls in the same reply
// are dropped -- the answer may change what to look up.
func askedQuestion(calls []assistant.ToolCall) (string, bool) {
	for _, c := range calls {
		if c.Name != askUserToolName {
			continue
		}
		var args struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal([]byte(c.Arguments), &args); err != nil || args.Question == "" {
			return c.Arguments, true
		}
		return args.Question, true
	}
	return "", false
}

// runTools runs each call outside any transaction (a tool's use case opens
// its own) and returns the step to record: the model's call message, then
// one result per call. Tool errors become results the model can react to.
func (uc *ProcessRequestUseCase) runTools(ctx context.Context, requestID string, reply assistant.ChatReply) []assistant.Message {
	step := []assistant.Message{{RequestID: requestID, Role: assistant.RoleAssistant, Content: reply.Content, ToolCalls: reply.ToolCalls}}
	for _, call := range reply.ToolCalls {
		step = append(step, assistant.Message{
			RequestID: requestID, Role: assistant.RoleTool, ToolCallID: call.ID, ToolName: call.Name,
			Content: truncate(uc.callTool(ctx, call), MaxToolResultChars),
		})
	}
	return step
}

func (uc *ProcessRequestUseCase) callTool(ctx context.Context, call assistant.ToolCall) string {
	tool, ok := uc.tools[call.Name]
	if !ok {
		return fmt.Sprintf(`{"error":"unknown tool %q"}`, call.Name)
	}
	args := json.RawMessage(call.Arguments)
	if !json.Valid(args) {
		args = json.RawMessage(`{}`)
	}
	out, err := tool.Call(ctx, args)
	if err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(b)
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…[truncated]"
}

// whileProcessing runs fn in a transaction with the request locked, only
// if it is still processing.
func (uc *ProcessRequestUseCase) whileProcessing(ctx context.Context, id string, fn func(ctx context.Context, r *assistant.Request) error) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.requests.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if r.Status != assistant.StatusProcessing {
			return errStopped
		}
		return fn(ctx, r)
	})
}

func (uc *ProcessRequestUseCase) appendWhileProcessing(ctx context.Context, id string, msgs []assistant.Message) error {
	return uc.whileProcessing(ctx, id, func(ctx context.Context, r *assistant.Request) error {
		return uc.appendAll(ctx, r, msgs)
	})
}

func (uc *ProcessRequestUseCase) appendAll(ctx context.Context, r *assistant.Request, msgs []assistant.Message) error {
	at := uc.now()
	for i := range msgs {
		msgs[i].RequestID, msgs[i].TenantID, msgs[i].CreatedAt = r.ID, r.TenantID, at
		if err := uc.messages.Append(ctx, &msgs[i]); err != nil {
			return err
		}
	}
	return nil
}

// finish records the turn's last message, moves the request on with
// transition, and raises the event -- atomically.
func (uc *ProcessRequestUseCase) finish(
	ctx context.Context, id string, msg assistant.Message,
	transition func(*assistant.Request, time.Time) error, action events.ActionType, data eventrecord.Data,
) error {
	return uc.whileProcessing(ctx, id, func(ctx context.Context, r *assistant.Request) error {
		if err := transition(r, uc.now()); err != nil {
			return err
		}
		if err := uc.requests.Update(ctx, r); err != nil {
			return err
		}
		if err := uc.appendAll(ctx, r, []assistant.Message{msg}); err != nil {
			return err
		}
		data["attempts"] = r.Attempts
		return eventrecord.Record(ctx, uc.outbox, action, r.ID, data)
	})
}

// retryLater puts the request back in the queue. Retry bookkeeping isn't a
// business action, so no event: the request's status and wait reason are
// what the UI shows.
func (uc *ProcessRequestUseCase) retryLater(ctx context.Context, id string, reason assistant.WaitReason, cause error) error {
	detail := ""
	if cause != nil {
		detail = truncate(cause.Error(), maxErrorDetail)
	}
	return uc.whileProcessing(ctx, id, func(ctx context.Context, r *assistant.Request) error {
		if err := r.RetryLater(reason, detail, uc.now()); err != nil {
			return err
		}
		return uc.requests.Update(ctx, r)
	})
}
