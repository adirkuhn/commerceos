package assistant

import (
	"encoding/json"
	"time"
)

// ProposalKind is a change the assistant can propose. Nothing is written
// until a person confirms it.
type ProposalKind string

const (
	KindCreateQuote  ProposalKind = "create_quote"
	KindConvertQuote ProposalKind = "convert_quote"
)

func (k ProposalKind) Valid() bool {
	return k == KindCreateQuote || k == KindConvertQuote
}

type ProposalStatus string

const (
	ProposalPending ProposalStatus = "pending"
	// ProposalExecuting: a confirm is running it -- a second confirm (a
	// double click, another tab) is refused instead of doing it twice.
	ProposalExecuting  ProposalStatus = "executing"
	ProposalConfirmed  ProposalStatus = "confirmed"
	ProposalRejected   ProposalStatus = "rejected"
	ProposalSuperseded ProposalStatus = "superseded"
)

// Proposal is one change waiting for a person's decision. Payload is the
// exact input the action runs with on confirm; Preview is what the person
// is shown (names, prices, totals), both JSON owned by the action.
type Proposal struct {
	ID        string
	TenantID  string
	RequestID string
	Kind      ProposalKind
	Status    ProposalStatus
	Payload   json.RawMessage
	Preview   json.RawMessage
	// LastError is why the last confirm failed; the proposal stays pending.
	LastError string
	// ResultID and ResultRef identify what a confirm created (the quote's
	// id and Q-0012, the order's id and number).
	ResultID  string
	ResultRef string
	DecidedBy string
	DecidedAt *time.Time
	CreatedAt time.Time
}

func NewProposal(tenantID, requestID string, kind ProposalKind, payload, preview json.RawMessage, now time.Time) (*Proposal, error) {
	if !kind.Valid() {
		return nil, ErrUnknownProposalKind
	}
	return &Proposal{
		TenantID: tenantID, RequestID: requestID, Kind: kind, Status: ProposalPending,
		Payload: payload, Preview: preview, CreatedAt: now,
	}, nil
}

// Claim starts a confirm.
func (p *Proposal) Claim(time.Time) error {
	if p.Status != ProposalPending {
		return ErrProposalNotPending
	}
	p.Status = ProposalExecuting
	return nil
}

// Release puts a proposal whose confirm failed back to pending with why.
func (p *Proposal) Release(reason string) error {
	if p.Status != ProposalExecuting {
		return ErrProposalNotPending
	}
	p.Status = ProposalPending
	p.LastError = reason
	return nil
}

// Complete records what the confirm created.
func (p *Proposal) Complete(by, resultID, resultRef string, now time.Time) error {
	if p.Status != ProposalExecuting {
		return ErrProposalNotPending
	}
	p.Status = ProposalConfirmed
	p.ResultID, p.ResultRef, p.LastError = resultID, resultRef, ""
	p.decide(by, now)
	return nil
}

func (p *Proposal) Reject(by string, now time.Time) error {
	if p.Status != ProposalPending {
		return ErrProposalNotPending
	}
	p.Status = ProposalRejected
	p.decide(by, now)
	return nil
}

// Supersede retires a pending proposal the conversation moved past (the
// user asked for changes, or cancelled the request).
func (p *Proposal) Supersede(now time.Time) error {
	if p.Status != ProposalPending {
		return ErrProposalNotPending
	}
	p.Status = ProposalSuperseded
	p.DecidedAt = &now
	return nil
}

func (p *Proposal) decide(by string, now time.Time) {
	p.DecidedBy = by
	p.DecidedAt = &now
}
