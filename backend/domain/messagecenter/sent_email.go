package messagecenter

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
)

// SentEmailSource is what sent an email.
type SentEmailSource string

const (
	SentEmailSourceAutomation   SentEmailSource = "automation"
	SentEmailSourceNotification SentEmailSource = "notification"
	SentEmailSourceSignInLink   SentEmailSource = "sign_in_link"
	SentEmailSourceNewsletter   SentEmailSource = "newsletter"
	// SentEmailSourceSupplierIssues: a purchase order's missing/damaged
	// items emailed to the supplier (procurement).
	SentEmailSourceSupplierIssues SentEmailSource = "supplier_issues"
)

func (s SentEmailSource) Valid() bool {
	switch s {
	case SentEmailSourceAutomation, SentEmailSourceNotification, SentEmailSourceSignInLink, SentEmailSourceNewsletter, SentEmailSourceSupplierIssues:
		return true
	}
	return false
}

// SentEmailStatus is what happened to an email. Sent means the mail server
// accepted it; delivery to the inbox isn't reported back over SMTP.
type SentEmailStatus string

const (
	SentEmailSent    SentEmailStatus = "sent"
	SentEmailFailed  SentEmailStatus = "failed"
	SentEmailSkipped SentEmailStatus = "skipped"
)

func (s SentEmailStatus) Valid() bool {
	return s == SentEmailSent || s == SentEmailFailed || s == SentEmailSkipped
}

// Why an automation email was skipped (SentEmail.Reason for SentEmailSkipped).
const (
	SkipNoRecipient      = "no_recipient"
	SkipTemplateDisabled = "template_disabled"
)

// SentEmail is one entry of the tenant's email history. Entries are kept
// until the tenant deletes them.
type SentEmail struct {
	ID        string
	TenantID  string
	Source    SentEmailSource
	Recipient string // empty for a skip with nobody to send to
	Subject   string
	HTMLBody  string
	Status    SentEmailStatus
	// Reason is the mail server's error for SentEmailFailed, a Skip* code
	// for SentEmailSkipped, empty for SentEmailSent.
	Reason string
	// RuleID, TemplateID and TriggerSubject are set for automation emails.
	RuleID         string
	TemplateID     string
	TriggerSubject string
	CreatedAt      time.Time
}

// SentEmailFilter narrows the history list; zero values don't filter.
type SentEmailFilter struct {
	Status SentEmailStatus
	Source SentEmailSource
	// Query matches recipient or subject, case-insensitively.
	Query   string
	From    *time.Time
	To      *time.Time
	Page    int
	PerPage int
}

const (
	defaultSentEmailsPerPage = 50
	maxSentEmailsPerPage     = 200
)

// Normalized fills in paging defaults and caps the page size.
func (f SentEmailFilter) Normalized() SentEmailFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = defaultSentEmailsPerPage
	}
	if f.PerPage > maxSentEmailsPerPage {
		f.PerPage = maxSentEmailsPerPage
	}
	return f
}

type SentEmailRepository interface {
	Create(ctx context.Context, e *SentEmail) error
	// List returns one page, newest first, without HTMLBody, and the total
	// number of entries matching the filter.
	List(ctx context.Context, f SentEmailFilter) ([]SentEmail, int, error)
	FindByID(ctx context.Context, id string) (*SentEmail, error)
	// Delete removes the given entries and returns the ids it removed.
	Delete(ctx context.Context, ids []string) ([]string, error)
}

var (
	ErrSentEmailNotFound = errs.New("SENT_EMAIL_NOT_FOUND", "sent email not found")
	ErrSentEmailFilter   = errs.New("SENT_EMAIL_INVALID_FILTER", "unknown sent email status or source")
	ErrNoSentEmailsGiven = errs.New("SENT_EMAIL_IDS_REQUIRED", "choose at least one sent email to delete")
)
