// Package assistant is the admin AI assistant: a person pastes or writes a
// request ("status of order CO-12?", "quote 10 boxes of farofa for Ana
// Bakery"), it waits in a durable queue until the tenant's AI model takes
// it, and the model answers by calling read-only tools that run as that
// person -- or asks a clarifying question. A request never leaves the
// queue because the model was busy or out of quota; only the user cancels
// it.
package assistant

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxTextLength  = 20000
	MaxTitleLength = 120

	firstRetryDelay = 30 * time.Second
	MaxRetryDelay   = 15 * time.Minute
)

type Status string

const (
	// StatusQueued: waiting for the worker (NextAttemptAt says when).
	StatusQueued Status = "queued"
	// StatusProcessing: the worker is running the model on it.
	StatusProcessing Status = "processing"
	// StatusNeedsInput: the model asked the user a question.
	StatusNeedsInput Status = "needs_input"
	// StatusAnswered: the model gave its final answer for this turn.
	StatusAnswered Status = "answered"
	// StatusAwaitingConfirmation: the model proposed a change (a quote, a
	// conversion) that waits for a person to confirm or reject it.
	StatusAwaitingConfirmation Status = "awaiting_confirmation"
	StatusCancelled            Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusQueued, StatusProcessing, StatusNeedsInput, StatusAnswered, StatusAwaitingConfirmation, StatusCancelled:
		return true
	}
	return false
}

// WaitReason says why a queued request is still waiting after an attempt.
type WaitReason string

const (
	WaitModelBusy           WaitReason = "model_busy"
	WaitNoProvider          WaitReason = "no_provider"
	WaitProviderUnsupported WaitReason = "provider_unsupported"
	WaitModelError          WaitReason = "model_error"
)

// Request is one conversation with the assistant. Its messages live in
// the message log (MessageRepository); the request is the queue entry and
// state machine.
type Request struct {
	ID       string
	TenantID string
	// UserID and UserRole are who asked: the worker runs every tool as
	// this person, so the assistant can never see more than they can.
	UserID        string
	UserRole      string
	Title         string
	Status        Status
	Attempts      int
	NextAttemptAt time.Time
	WaitReason    WaitReason
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewRequest validates the opening text and queues the request, due now.
func NewRequest(tenantID, userID, userRole, text string, now time.Time) (*Request, error) {
	if err := ValidateText(text); err != nil {
		return nil, err
	}
	return &Request{
		TenantID: tenantID, UserID: userID, UserRole: userRole, Title: titleOf(text),
		Status: StatusQueued, NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ValidateText guards both the opening text and every reply.
func ValidateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return ErrTextRequired
	}
	if utf8.RuneCountInString(text) > MaxTextLength {
		return ErrTextTooLong
	}
	return nil
}

func titleOf(text string) string {
	line := strings.TrimSpace(text)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	runes := []rune(line)
	if len(runes) > MaxTitleLength {
		return string(runes[:MaxTitleLength-1]) + "…"
	}
	return line
}

// RetryDelay is the wait after the given (1-based) failed attempt:
// 30s doubling up to MaxRetryDelay.
func RetryDelay(attempt int) time.Duration {
	d := firstRetryDelay
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= MaxRetryDelay {
			return MaxRetryDelay
		}
	}
	return d
}

// Start claims the request for one worker attempt.
func (r *Request) Start(now time.Time) error {
	if r.Status != StatusQueued {
		return ErrInvalidTransition
	}
	r.Status = StatusProcessing
	r.Attempts++
	r.UpdatedAt = now
	return nil
}

// RetryLater puts a request whose attempt failed back in the queue with
// backoff -- it never gives up on its own.
func (r *Request) RetryLater(reason WaitReason, detail string, now time.Time) error {
	if r.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	r.Status = StatusQueued
	r.WaitReason = reason
	r.LastError = detail
	r.NextAttemptAt = now.Add(RetryDelay(r.Attempts))
	r.UpdatedAt = now
	return nil
}

// Answer records that the model finished this turn.
func (r *Request) Answer(now time.Time) error {
	return r.finishTurn(StatusAnswered, now)
}

// AskUser records that the model needs the user to clarify.
func (r *Request) AskUser(now time.Time) error {
	return r.finishTurn(StatusNeedsInput, now)
}

func (r *Request) finishTurn(to Status, now time.Time) error {
	if r.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	r.Status = to
	r.WaitReason = ""
	r.LastError = ""
	r.UpdatedAt = now
	return nil
}

// Propose records that the model proposed a change for the user to decide.
func (r *Request) Propose(now time.Time) error {
	return r.finishTurn(StatusAwaitingConfirmation, now)
}

// Decided records that the user confirmed or rejected the pending
// proposal: the request keeps waiting when a follow-up was proposed (a
// created quote offering its conversion), else the turn is answered.
func (r *Request) Decided(followUp bool, now time.Time) error {
	if r.Status != StatusAwaitingConfirmation {
		return ErrInvalidTransition
	}
	if !followUp {
		r.Status = StatusAnswered
	}
	r.UpdatedAt = now
	return nil
}

// Reply queues the user's next message: an answer to the model's question,
// a follow-up on its answer, or changes to what it proposed. A new turn
// starts its own backoff.
func (r *Request) Reply(now time.Time) error {
	if r.Status != StatusNeedsInput && r.Status != StatusAnswered && r.Status != StatusAwaitingConfirmation {
		return ErrInvalidTransition
	}
	r.Status = StatusQueued
	r.Attempts = 0
	r.NextAttemptAt = now
	r.UpdatedAt = now
	return nil
}

// Cancel stops a request that hasn't finished: queued, processing, or
// waiting for the user (an answer or a decision).
func (r *Request) Cancel(now time.Time) error {
	switch r.Status {
	case StatusQueued, StatusProcessing, StatusNeedsInput, StatusAwaitingConfirmation:
	default:
		return ErrNotCancellable
	}
	r.Status = StatusCancelled
	r.UpdatedAt = now
	return nil
}
