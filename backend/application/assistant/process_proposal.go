package assistant

import (
	"context"
	"encoding/json"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/events"
)

// proposalCall finds the first call to a proposal action's tool; any
// other calls in the same reply are dropped -- a proposal ends the turn.
func (uc *ProcessRequestUseCase) proposalCall(calls []assistant.ToolCall) (assistant.ToolCall, ProposalAction, bool) {
	for _, c := range calls {
		if a, ok := uc.actions[c.Name]; ok {
			return c, a, true
		}
	}
	return assistant.ToolCall{}, nil, false
}

// rejectedProposalStep records a proposal Prepare refused: the error goes
// back to the model as the call's result, so it can correct itself.
func rejectedProposalStep(requestID, content string, call assistant.ToolCall, err error) []assistant.Message {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return []assistant.Message{
		{RequestID: requestID, Role: assistant.RoleAssistant, Content: content, ToolCalls: []assistant.ToolCall{call}},
		{RequestID: requestID, Role: assistant.RoleTool, ToolCallID: call.ID, ToolName: call.Name, Content: string(b)},
	}
}

// proposedResult is what the model reads as the proposal tool's result.
const proposedNote = "Shown to the user; waiting for their confirmation. Nothing has been created yet; don't propose it again."

// propose stores the prepared proposal and waits for a person to decide.
func (uc *ProcessRequestUseCase) propose(
	ctx context.Context, requestID, content string, call assistant.ToolCall,
	kind assistant.ProposalKind, payload, preview json.RawMessage,
) error {
	return uc.whileProcessing(ctx, requestID, func(ctx context.Context, r *assistant.Request) error {
		p, err := assistant.NewProposal(r.TenantID, r.ID, kind, payload, preview, uc.now())
		if err != nil {
			return err
		}
		if err := uc.proposals.Create(ctx, p); err != nil {
			return err
		}
		if err := r.Propose(uc.now()); err != nil {
			return err
		}
		if err := uc.requests.Update(ctx, r); err != nil {
			return err
		}
		result, _ := json.Marshal(map[string]string{"status": "proposed", "proposal_id": p.ID, "note": proposedNote})
		if err := uc.appendAll(ctx, r, []assistant.Message{
			{Role: assistant.RoleAssistant, Content: content, ToolCalls: []assistant.ToolCall{call}},
			{Role: assistant.RoleTool, ToolCallID: call.ID, ToolName: call.Name, Content: string(result),
				Notice: assistant.NoticeProposed, ProposalID: p.ID},
		}); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AssistantProposalCreated, p.ID,
			eventrecord.Data{"request_id": r.ID, "kind": string(kind)})
	})
}
