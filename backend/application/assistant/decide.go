package assistant

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DecideDeps are the ports confirming and rejecting a proposal need.
type DecideDeps struct {
	Requests  assistant.RequestRepository
	Messages  assistant.MessageRepository
	Proposals assistant.ProposalRepository
	Actions   []ProposalAction
	Outbox    eventrecord.Enqueuer
	Now       func() time.Time
}

func actionsByKind(actions []ProposalAction) map[assistant.ProposalKind]ProposalAction {
	out := make(map[assistant.ProposalKind]ProposalAction, len(actions))
	for _, a := range actions {
		out[a.Kind()] = a
	}
	return out
}

// lockPending locks the request and its pending proposal, checking the
// request is waiting on exactly this proposal's decision.
func lockPending(ctx context.Context, d DecideDeps, requestID, proposalID string) (*assistant.Request, *assistant.Proposal, error) {
	r, err := d.Requests.GetForUpdate(ctx, requestID)
	if err != nil {
		return nil, nil, err
	}
	p, err := d.Proposals.GetForUpdate(ctx, proposalID)
	if err != nil {
		return nil, nil, err
	}
	if p.RequestID != r.ID {
		return nil, nil, assistant.ErrProposalNotFound
	}
	if p.Status != assistant.ProposalPending {
		return nil, nil, assistant.ErrProposalNotPending
	}
	if r.Status != assistant.StatusAwaitingConfirmation {
		return nil, nil, assistant.ErrInvalidTransition
	}
	return r, p, nil
}

// ConfirmProposalUseCase runs a pending proposal as the confirming person
// -- their own permissions apply, through the action's use case, which
// raises its own event (quotes.quote.created, ...). A follow-up the action
// offers (a created quote's conversion) is proposed next.
type ConfirmProposalUseCase struct {
	db      *gorm.DB
	d       DecideDeps
	actions map[assistant.ProposalKind]ProposalAction
}

func NewConfirmProposalUseCase(db *gorm.DB, d DecideDeps) *ConfirmProposalUseCase {
	return &ConfirmProposalUseCase{db: db, d: d, actions: actionsByKind(d.Actions)}
}

func (uc *ConfirmProposalUseCase) Execute(ctx context.Context, requestID, proposalID string, choices ConfirmChoices) (*RequestDetail, error) {
	var claimed assistant.Proposal
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		_, p, err := lockPending(ctx, uc.d, requestID, proposalID)
		if err != nil {
			return err
		}
		if err := p.Claim(uc.d.Now()); err != nil {
			return err
		}
		claimed = *p
		return uc.d.Proposals.Update(ctx, p)
	})
	if err != nil {
		return nil, fmt.Errorf("confirmAssistantProposal: %w", err)
	}

	result, err := uc.run(ctx, claimed, choices)
	if err != nil {
		if relErr := uc.release(ctx, proposalID, err); relErr != nil {
			log.Printf("assistant: release proposal %s after failed confirm: %v", proposalID, relErr)
		}
		return nil, fmt.Errorf("confirmAssistantProposal: %w", err)
	}
	next := uc.prepareFollowUp(ctx, result.FollowUp)

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.record(ctx, requestID, proposalID, result, next)
	})
	if err != nil {
		return nil, fmt.Errorf("confirmAssistantProposal: record %s: %w", result.Ref, err)
	}
	return loadDetail(ctx, uc.db, uc.d.Requests, uc.d.Messages, uc.d.Proposals, requestID)
}

func (uc *ConfirmProposalUseCase) run(ctx context.Context, p assistant.Proposal, choices ConfirmChoices) (ActionResult, error) {
	action, ok := uc.actions[p.Kind]
	if !ok {
		return ActionResult{}, assistant.ErrUnknownProposalKind
	}
	return action.Execute(ctx, p.Payload, choices)
}

func (uc *ConfirmProposalUseCase) release(ctx context.Context, proposalID string, cause error) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.d.Proposals.GetForUpdate(ctx, proposalID)
		if err != nil {
			return err
		}
		if err := p.Release(truncate(cause.Error(), maxErrorDetail)); err != nil {
			return err
		}
		return uc.d.Proposals.Update(ctx, p)
	})
}

type preparedFollowUp struct {
	kind             assistant.ProposalKind
	payload, preview []byte
}

// prepareFollowUp runs the follow-up's Prepare. A follow-up that can't be
// prepared (no warehouse yet, say) is skipped: what was confirmed stays
// done, and the person can still ask for the next step in a reply.
func (uc *ConfirmProposalUseCase) prepareFollowUp(ctx context.Context, f *FollowUp) *preparedFollowUp {
	if f == nil {
		return nil
	}
	action, ok := uc.actions[f.Kind]
	if !ok {
		return nil
	}
	payload, preview, err := action.Prepare(ctx, f.Args)
	if err != nil {
		log.Printf("assistant: follow-up %s not offered: %v", f.Kind, err)
		return nil
	}
	return &preparedFollowUp{kind: f.Kind, payload: payload, preview: preview}
}

// record completes the proposal, offers the follow-up, and moves the
// request on -- unless the request was cancelled while the action ran: the
// result is still recorded, but nothing more is proposed.
func (uc *ConfirmProposalUseCase) record(ctx context.Context, requestID, proposalID string, result ActionResult, next *preparedFollowUp) error {
	at := uc.d.Now()
	r, err := uc.d.Requests.GetForUpdate(ctx, requestID)
	if err != nil {
		return err
	}
	p, err := uc.d.Proposals.GetForUpdate(ctx, proposalID)
	if err != nil {
		return err
	}
	if err := p.Complete(tenant.UserIDFromContext(ctx), result.ID, result.Ref, at); err != nil {
		return err
	}
	if err := uc.d.Proposals.Update(ctx, p); err != nil {
		return err
	}
	if err := uc.d.Messages.Append(ctx, &assistant.Message{
		RequestID: r.ID, TenantID: r.TenantID, Role: assistant.RoleAssistant, CreatedAt: at,
		Notice: assistant.NoticeProposalConfirmed, ProposalID: p.ID,
		Content: fmt.Sprintf("The user confirmed the %s proposal: created %s (id %s).", p.Kind, result.Ref, result.ID),
	}); err != nil {
		return err
	}
	if err := eventrecord.Record(ctx, uc.d.Outbox, events.AssistantProposalConfirmed, p.ID, eventrecord.Data{
		"request_id": r.ID, "kind": string(p.Kind), "result_id": result.ID, "result_ref": result.Ref,
	}); err != nil {
		return err
	}
	if r.Status != assistant.StatusAwaitingConfirmation {
		return nil
	}
	if next != nil {
		follow, err := assistant.NewProposal(r.TenantID, r.ID, next.kind, next.payload, next.preview, at)
		if err != nil {
			return err
		}
		if err := uc.d.Proposals.Create(ctx, follow); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.d.Outbox, events.AssistantProposalCreated, follow.ID,
			eventrecord.Data{"request_id": r.ID, "kind": string(follow.Kind)}); err != nil {
			return err
		}
	}
	if err := r.Decided(next != nil, at); err != nil {
		return err
	}
	return uc.d.Requests.Update(ctx, r)
}

// RejectProposalUseCase records that the person doesn't want the change.
type RejectProposalUseCase struct {
	db *gorm.DB
	d  DecideDeps
}

func NewRejectProposalUseCase(db *gorm.DB, d DecideDeps) *RejectProposalUseCase {
	return &RejectProposalUseCase{db: db, d: d}
}

func (uc *RejectProposalUseCase) Execute(ctx context.Context, requestID, proposalID string) (*RequestDetail, error) {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, p, err := lockPending(ctx, uc.d, requestID, proposalID)
		if err != nil {
			return err
		}
		at := uc.d.Now()
		if err := p.Reject(tenant.UserIDFromContext(ctx), at); err != nil {
			return err
		}
		if err := uc.d.Proposals.Update(ctx, p); err != nil {
			return err
		}
		if err := r.Decided(false, at); err != nil {
			return err
		}
		if err := uc.d.Requests.Update(ctx, r); err != nil {
			return err
		}
		if err := uc.d.Messages.Append(ctx, &assistant.Message{
			RequestID: r.ID, TenantID: r.TenantID, Role: assistant.RoleAssistant, CreatedAt: at,
			Notice: assistant.NoticeProposalRejected, ProposalID: p.ID,
			Content: fmt.Sprintf("The user rejected the %s proposal; nothing was created.", p.Kind),
		}); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.d.Outbox, events.AssistantProposalRejected, p.ID,
			eventrecord.Data{"request_id": r.ID, "kind": string(p.Kind)})
	})
	if err != nil {
		return nil, fmt.Errorf("rejectAssistantProposal: %w", err)
	}
	return loadDetail(ctx, uc.db, uc.d.Requests, uc.d.Messages, uc.d.Proposals, requestID)
}
