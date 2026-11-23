package assistant

import (
	"context"
	"encoding/json"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// ProposalAction is one kind of change the assistant can propose (a
// strategy: a new kind is a new type, registered where the worker and the
// API are wired). The model calls its Tool; Prepare turns the model's
// arguments into a checked, priced proposal without writing anything;
// Execute does the change when a person confirms.
type ProposalAction interface {
	Kind() assistant.ProposalKind
	// Tool is what the model calls to propose this change.
	Tool() assistant.ToolDefinition
	// Prepare validates args against current data and returns the exact
	// input Execute will run with (payload) and what the person is shown
	// (preview). An error goes back to the model so it can correct itself.
	// It runs as the requesting user.
	Prepare(ctx context.Context, args json.RawMessage) (payload, preview json.RawMessage, err error)
	// Execute runs payload as the confirming user (their permissions
	// apply), with the choices they made on the confirm screen.
	Execute(ctx context.Context, payload json.RawMessage, choices ConfirmChoices) (ActionResult, error)
}

// ConfirmChoices are what the person can still pick when confirming.
type ConfirmChoices struct {
	// WarehouseID overrides the proposed warehouse (quote conversion).
	WarehouseID string
}

// ActionResult is what a confirmed action created.
type ActionResult struct {
	ID  string
	Ref string
	// FollowUp, when set, is proposed next in the same request (a created
	// quote offers its conversion to an order).
	FollowUp *FollowUp
}

// FollowUp is the next proposal: Args goes through that kind's Prepare.
type FollowUp struct {
	Kind assistant.ProposalKind
	Args json.RawMessage
}
