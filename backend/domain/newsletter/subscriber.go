// Package newsletter is a store's mailing list with double opt-in: a
// signup is only a subscription once the address owner confirms it from
// the email we send (GDPR-proof consent), and every subscriber can
// unsubscribe with their own link.
package newsletter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrInvalidEmail       = errors.New("enter a valid email address")
	ErrConsentRequired    = errors.New("consent is required to subscribe")
	ErrInvalidToken       = errors.New("this link is invalid or has expired")
	ErrSubscriberNotFound = errors.New("subscriber not found")
)

type Status string

const (
	StatusPending      Status = "pending"
	StatusConfirmed    Status = "confirmed"
	StatusUnsubscribed Status = "unsubscribed"
)

const (
	// ConfirmationTTL is how long a confirmation link works.
	ConfirmationTTL = 7 * 24 * time.Hour
	// ResendAfter throttles confirmation emails to one address, so the
	// signup form can't be used to flood someone's inbox.
	ResendAfter    = 10 * time.Minute
	maxEmailLength = 254
)

type Subscriber struct {
	ID               string
	TenantID         string
	SalesChannelID   string
	Email            string
	Status           Status
	ConfirmTokenHash string
	// UnsubscribeToken is kept in the clear: it goes into every email we
	// send and only ever allows unsubscribing.
	UnsubscribeToken   string
	ConsentAt          time.Time
	ConfirmationSentAt time.Time
	ConfirmedAt        *time.Time
	UnsubscribedAt     *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > maxEmailLength {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" || !strings.Contains(email[strings.Index(email, "@")+1:], ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func NewSubscriber(tenantID, channelID, email, confirmToken, unsubscribeToken string, now time.Time) Subscriber {
	return Subscriber{
		TenantID: tenantID, SalesChannelID: channelID, Email: email, Status: StatusPending,
		ConfirmTokenHash: HashToken(confirmToken), UnsubscribeToken: unsubscribeToken,
		ConsentAt: now, ConfirmationSentAt: now, CreatedAt: now, UpdatedAt: now,
	}
}

// Resubscribe handles a signup for an address we already know. It reports
// whether a (new) confirmation email should be sent: never for a confirmed
// subscriber, at most every ResendAfter for a pending one.
func (s *Subscriber) Resubscribe(confirmToken string, now time.Time) bool {
	switch s.Status {
	case StatusConfirmed:
		return false
	case StatusPending:
		if now.Sub(s.ConfirmationSentAt) < ResendAfter {
			return false
		}
	case StatusUnsubscribed:
		s.Status = StatusPending
		s.UnsubscribedAt = nil
		s.ConfirmedAt = nil
	}
	s.ConfirmTokenHash = HashToken(confirmToken)
	s.ConsentAt, s.ConfirmationSentAt, s.UpdatedAt = now, now, now
	return true
}

// Confirm completes the opt-in. The link works once and expires.
func (s *Subscriber) Confirm(now time.Time) error {
	if s.Status != StatusPending || s.ConfirmTokenHash == "" || now.Sub(s.ConfirmationSentAt) > ConfirmationTTL {
		return ErrInvalidToken
	}
	s.Status = StatusConfirmed
	s.ConfirmedAt = &now
	s.ConfirmTokenHash = ""
	s.UpdatedAt = now
	return nil
}

func (s *Subscriber) Unsubscribe(now time.Time) {
	s.Status = StatusUnsubscribed
	s.UnsubscribedAt = &now
	s.ConfirmTokenHash = ""
	s.UpdatedAt = now
}

type ListFilter struct {
	Status *Status
	Search string
	Limit  int
	Offset int
}

type Repository interface {
	// Create fills in s.ID.
	Create(ctx context.Context, s *Subscriber) error
	// FindByEmail / FindByConfirmTokenHash / FindByUnsubscribeToken return
	// nil, nil when nothing matches.
	FindByEmail(ctx context.Context, email string) (*Subscriber, error)
	FindByConfirmTokenHash(ctx context.Context, hash string) (*Subscriber, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (*Subscriber, error)
	Update(ctx context.Context, s *Subscriber) error
	List(ctx context.Context, f ListFilter) ([]Subscriber, int64, error)
	// Delete erases the subscriber entirely (right to erasure);
	// ErrSubscriberNotFound when missing.
	Delete(ctx context.Context, id string) error
}
