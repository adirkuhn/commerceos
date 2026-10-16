package procurement

import (
	"context"
	"fmt"
	"html"
	"net/mail"
	"strings"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/messaging"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// EmailSupplierIssuesInput is a message written (and editable) in the
// admin -- the issue list in the reader's language -- plus the issues it
// reports, which become claimed once it is sent.
type EmailSupplierIssuesInput struct {
	PurchaseOrderID string
	To              string
	Subject         string
	// Body is plain text; it is escaped, never sent as HTML.
	Body string
	Refs []procurementDomain.SupplierIssueRef
}

type EmailSupplierIssuesResult struct {
	SentTo string
	// Claimed is how many of the reported issues went from open to
	// claimed; claimed or resolved ones keep their status.
	Claimed int
}

// EmailSupplierIssuesUseCase emails a PO's issue list to the supplier with
// the tenant's email connection, then marks the reported open issues
// claimed. The send happens between two transactions: never inside one,
// and nothing is claimed when it fails.
type EmailSupplierIssuesUseCase struct {
	db     *gorm.DB
	deps   SupplierIssueDeps
	mailer messaging.Client
}

func NewEmailSupplierIssuesUseCase(db *gorm.DB, deps SupplierIssueDeps, mailer messaging.Client) *EmailSupplierIssuesUseCase {
	return &EmailSupplierIssuesUseCase{db: db, deps: deps, mailer: mailer}
}

func (uc *EmailSupplierIssuesUseCase) Execute(ctx context.Context, in EmailSupplierIssuesInput) (*EmailSupplierIssuesResult, error) {
	if err := validateSupplierEmail(in); err != nil {
		return nil, fmt.Errorf("emailSupplierIssues: %w", err)
	}
	to := strings.TrimSpace(in.To)

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		_, err := uc.reportedIssues(ctx, in)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("emailSupplierIssues: %w", err)
	}

	if err := uc.mailer.Send(ctx, messaging.Message{
		Channel: messaging.ChannelEmail, To: to, Subject: strings.TrimSpace(in.Subject), HTMLBody: plainTextEmailHTML(in.Body),
	}); err != nil {
		return nil, fmt.Errorf("emailSupplierIssues: %w: %w", procurementDomain.ErrSupplierEmailNotSent, err)
	}

	result := &EmailSupplierIssuesResult{SentTo: to}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		// Re-read: a status may have changed while the email was on its way.
		issues, err := uc.reportedIssues(ctx, in)
		if err != nil {
			return err
		}
		for _, issue := range issues {
			if issue.Status != procurementDomain.SupplierIssueStatusOpen {
				continue
			}
			if _, err := uc.deps.setStatus(ctx, in.PurchaseOrderID, issue, procurementDomain.SupplierIssueStatusClaimed, issue.StatusNote); err != nil {
				return err
			}
			result.Claimed++
		}
		return eventrecord.Record(ctx, uc.deps.Outbox, events.SupplierIssueReportEmailed, in.PurchaseOrderID, eventrecord.Data{
			"to": to, "issue_count": len(issues), "claimed_count": result.Claimed,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("emailSupplierIssues: email sent but issues not marked claimed: %w", err)
	}
	return result, nil
}

// reportedIssues returns in.Refs' issues; every ref must be an issue of
// the PO. Call inside tenant.WithTenant.
func (uc *EmailSupplierIssuesUseCase) reportedIssues(ctx context.Context, in EmailSupplierIssuesInput) ([]procurementDomain.SupplierIssue, error) {
	if _, err := uc.deps.PurchaseOrders.FindByID(ctx, in.PurchaseOrderID); err != nil {
		return nil, err
	}
	all, err := uc.deps.collect(ctx, in.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	out := make([]procurementDomain.SupplierIssue, 0, len(in.Refs))
	for _, ref := range in.Refs {
		issue, ok := procurementDomain.FindSupplierIssue(all, ref)
		if !ok {
			return nil, procurementDomain.ErrSupplierIssueNotFound
		}
		out = append(out, issue)
	}
	return out, nil
}

func validateSupplierEmail(in EmailSupplierIssuesInput) error {
	addr, err := mail.ParseAddress(strings.TrimSpace(in.To))
	if err != nil || addr.Address != strings.TrimSpace(in.To) {
		return procurementDomain.ErrInvalidSupplierEmail
	}
	if strings.TrimSpace(in.Subject) == "" || strings.TrimSpace(in.Body) == "" {
		return procurementDomain.ErrSupplierEmailIncomplete
	}
	if len(in.Refs) == 0 {
		return procurementDomain.ErrNoSupplierIssuesSelected
	}
	return nil
}

// plainTextEmailHTML keeps the typed line breaks and spacing of a plain
// text message in an HTML email.
func plainTextEmailHTML(body string) string {
	return `<div style="font-family:sans-serif;white-space:pre-wrap">` + html.EscapeString(body) + `</div>`
}
