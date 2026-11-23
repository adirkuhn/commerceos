package assistant

import (
	"context"
	"encoding/json"
	"time"
)

// RequestRepository stores requests; the transaction comes from ctx.
type RequestRepository interface {
	Create(ctx context.Context, r *Request) error
	Get(ctx context.Context, id string) (*Request, error)
	// GetForUpdate is Get with the row locked until the transaction ends,
	// so a cancel and the worker finishing an attempt never overwrite
	// each other.
	GetForUpdate(ctx context.Context, id string) (*Request, error)
	// List returns the newest first; cursor is the last ID of the previous
	// page ("" for the first).
	List(ctx context.Context, f ListFilter) ([]Request, error)
	Update(ctx context.Context, r *Request) error
	// LockDue returns up to limit queued requests due at now, across every
	// tenant (the caller runs as super admin), locked FOR UPDATE until the
	// transaction ends and skipping rows another worker has locked -- the
	// caller Starts and Updates them in the same transaction.
	LockDue(ctx context.Context, now time.Time, limit int) ([]Request, error)
	// RequeueStale puts requests stuck in processing since before cutoff
	// (a worker that died mid-attempt) back in the queue.
	RequeueStale(ctx context.Context, cutoff, now time.Time) (int64, error)
}

type ListFilter struct {
	Status Status
	Cursor string
	Limit  int
}

// MessageRepository is a request's conversation log.
type MessageRepository interface {
	Append(ctx context.Context, m *Message) error
	ListByRequest(ctx context.Context, requestID string) ([]Message, error)
}

// ToolDefinition is what the model sees of a tool: Parameters is a JSON
// schema object.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Tool is one read-only capability the model may call. Call runs as the
// requesting user (tenant, user and role in ctx) and returns a JSON
// result for the model; a returned error is shown to the model as the
// tool's result, not treated as a failed attempt.
type Tool interface {
	Definition() ToolDefinition
	Call(ctx context.Context, args json.RawMessage) (string, error)
}

// ChatTurn is one model call: the instructions, the conversation so far,
// and the tools it may call.
type ChatTurn struct {
	System   string
	Messages []Message
	Tools    []ToolDefinition
}

// ChatReply is the model's next step: tool calls to run, or final text.
type ChatReply struct {
	Content   string
	ToolCalls []ToolCall
}

// ChatModel runs one step of the conversation. A transient failure (rate
// limit, quota, overload, timeout) wraps ErrModelBusy.
type ChatModel interface {
	Next(ctx context.Context, turn ChatTurn) (ChatReply, error)
}

// ChatModelResolver finds the tenant's configured model: nil, nil when no
// AI provider is connected; ErrProviderUnsupported when the connected one
// can't run the assistant.
type ChatModelResolver interface {
	Resolve(ctx context.Context) (ChatModel, error)
}
