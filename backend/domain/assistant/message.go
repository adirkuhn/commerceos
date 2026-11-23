package assistant

import "time"

type Role string

const (
	// RoleUser: what the person wrote (the opening text or a reply).
	RoleUser Role = "user"
	// RoleAssistant: the model's text and/or the tool calls it made.
	RoleAssistant Role = "assistant"
	// RoleTool: one tool's result, answering the call with ToolCallID.
	RoleTool Role = "tool"
)

// ToolCall is one tool the model asked to run, with its JSON arguments.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Message is one entry of a request's conversation, in order (Seq).
type Message struct {
	ID        string
	RequestID string
	TenantID  string
	Seq       int
	Role      Role
	Content   string
	ToolCalls []ToolCall
	// ToolCallID and ToolName are set on RoleTool messages.
	ToolCallID string
	ToolName   string
	// Question marks the assistant message that asked the user to clarify.
	Question bool
	// Notice is set instead of Content when the app, not the model, wrote
	// the message -- clients show it translated.
	Notice Notice
	// ProposalID links the message to the proposal it made or decided.
	ProposalID string
	CreatedAt  time.Time
}

type Notice string

// NoticeTooManySteps: the model kept calling tools without answering, so
// the turn stopped and the user is asked to add detail.
const NoticeTooManySteps Notice = "too_many_steps"

// NoticeProposed, NoticeProposalConfirmed and NoticeProposalRejected mark
// the messages recording a proposal and the person's decision on it
// (ProposalID says which). Their Content is the plain-English record the
// model reads on later turns.
const (
	NoticeProposed          Notice = "proposed"
	NoticeProposalConfirmed Notice = "proposal_confirmed"
	NoticeProposalRejected  Notice = "proposal_rejected"
)
