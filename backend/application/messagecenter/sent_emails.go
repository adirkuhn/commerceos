package messagecenter

import (
	"context"
	"fmt"
	"html"
	"log"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RecordSentEmailUseCase adds one entry to the tenant's email history.
type RecordSentEmailUseCase struct {
	db     *gorm.DB
	repo   mcDomain.SentEmailRepository
	outbox eventrecord.Enqueuer
}

func NewRecordSentEmailUseCase(db *gorm.DB, repo mcDomain.SentEmailRepository, outbox eventrecord.Enqueuer) *RecordSentEmailUseCase {
	return &RecordSentEmailUseCase{db: db, repo: repo, outbox: outbox}
}

// Execute records e for the tenant in ctx. Without a tenant (a platform
// email) there is no history to add it to: it logs and records nothing.
func (uc *RecordSentEmailUseCase) Execute(ctx context.Context, e mcDomain.SentEmail) error {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		log.Printf("recordSentEmail: %s email %q has no tenant, not added to any history", e.Source, e.Subject)
		return nil
	}
	e.TenantID = tenantID
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Create(ctx, &e); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SentEmailRecorded, e.ID, eventrecord.Data{
			"source": string(e.Source), "status": string(e.Status), "trigger_subject": e.TriggerSubject,
		})
	})
	if err != nil {
		return fmt.Errorf("recordSentEmail: %w", err)
	}
	return nil
}

// sentEmailRecorder is RecordSentEmailUseCase narrowed to point of use.
type sentEmailRecorder interface {
	Execute(ctx context.Context, e mcDomain.SentEmail) error
}

// RecordingEmailClient is a messaging.Client that adds every email it
// sends (or fails to send) to the tenant's history, as source. Wrap the
// email client of any module with it; the history is best effort -- a
// failure to record is logged and never fails the send.
type RecordingEmailClient struct {
	inner    messaging.Client
	recorder sentEmailRecorder
	source   mcDomain.SentEmailSource
}

func NewRecordingEmailClient(inner messaging.Client, recorder sentEmailRecorder, source mcDomain.SentEmailSource) *RecordingEmailClient {
	return &RecordingEmailClient{inner: inner, recorder: recorder, source: source}
}

var _ messaging.Client = (*RecordingEmailClient)(nil)

func (c *RecordingEmailClient) Send(ctx context.Context, msg messaging.Message) error {
	sendErr := c.inner.Send(ctx, msg)
	entry := mcDomain.SentEmail{Source: c.source, Recipient: msg.To, Subject: msg.Subject, HTMLBody: historyBody(msg), Status: mcDomain.SentEmailSent}
	if sendErr != nil {
		entry.Status, entry.Reason = mcDomain.SentEmailFailed, sendErr.Error()
	}
	recordBestEffort(ctx, c.recorder, entry)
	return sendErr
}

// historyBody is what the history shows: the HTML, or a plain-text body
// kept readable as HTML.
func historyBody(msg messaging.Message) string {
	if msg.HTMLBody != "" || msg.Body == "" {
		return msg.HTMLBody
	}
	return "<pre>" + html.EscapeString(msg.Body) + "</pre>"
}

func recordBestEffort(ctx context.Context, recorder sentEmailRecorder, e mcDomain.SentEmail) {
	if recorder == nil {
		return
	}
	if err := recorder.Execute(ctx, e); err != nil {
		log.Printf("sent email history: %s email to %s not recorded: %v", e.Source, e.Recipient, err)
	}
}

// SentEmailPage is one page of the history.
type SentEmailPage struct {
	Items   []mcDomain.SentEmail
	Total   int
	Page    int
	PerPage int
}

type ListSentEmailsUseCase struct {
	db   *gorm.DB
	repo mcDomain.SentEmailRepository
}

func NewListSentEmailsUseCase(db *gorm.DB, repo mcDomain.SentEmailRepository) *ListSentEmailsUseCase {
	return &ListSentEmailsUseCase{db: db, repo: repo}
}

func (uc *ListSentEmailsUseCase) Execute(ctx context.Context, f mcDomain.SentEmailFilter) (*SentEmailPage, error) {
	if (f.Status != "" && !f.Status.Valid()) || (f.Source != "" && !f.Source.Valid()) {
		return nil, fmt.Errorf("listSentEmails: %w", mcDomain.ErrSentEmailFilter)
	}
	f = f.Normalized()
	page := SentEmailPage{Page: f.Page, PerPage: f.PerPage}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		page.Items, page.Total, err = uc.repo.List(ctx, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSentEmails: %w", err)
	}
	return &page, nil
}

type GetSentEmailUseCase struct {
	db   *gorm.DB
	repo mcDomain.SentEmailRepository
}

func NewGetSentEmailUseCase(db *gorm.DB, repo mcDomain.SentEmailRepository) *GetSentEmailUseCase {
	return &GetSentEmailUseCase{db: db, repo: repo}
}

func (uc *GetSentEmailUseCase) Execute(ctx context.Context, id string) (*mcDomain.SentEmail, error) {
	var e *mcDomain.SentEmail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		e, err = uc.repo.FindByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getSentEmail: %w", err)
	}
	return e, nil
}

// DeleteSentEmailsUseCase removes entries from the history -- the only
// way they go: the history is kept until the tenant deletes it.
type DeleteSentEmailsUseCase struct {
	db     *gorm.DB
	repo   mcDomain.SentEmailRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteSentEmailsUseCase(db *gorm.DB, repo mcDomain.SentEmailRepository, outbox eventrecord.Enqueuer) *DeleteSentEmailsUseCase {
	return &DeleteSentEmailsUseCase{db: db, repo: repo, outbox: outbox}
}

// Execute deletes ids and returns how many it removed. Ids already gone
// are ignored, except a single one, which is ErrSentEmailNotFound.
func (uc *DeleteSentEmailsUseCase) Execute(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("deleteSentEmails: %w", mcDomain.ErrNoSentEmailsGiven)
	}
	var deleted []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if deleted, err = uc.repo.Delete(ctx, ids); err != nil {
			return err
		}
		if len(ids) == 1 && len(deleted) == 0 {
			return mcDomain.ErrSentEmailNotFound
		}
		for _, id := range deleted {
			if err := eventrecord.Record(ctx, uc.outbox, events.SentEmailDeleted, id, eventrecord.Data{}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("deleteSentEmails: %w", err)
	}
	return len(deleted), nil
}

// RecordingEmailClientFactory makes clients that record what they send
// (see RecordingEmailClient) -- for modules that build a client per
// tenant email connection.
type RecordingEmailClientFactory struct {
	inner    messaging.EmailClientFactory
	recorder sentEmailRecorder
	source   mcDomain.SentEmailSource
}

func NewRecordingEmailClientFactory(inner messaging.EmailClientFactory, recorder sentEmailRecorder, source mcDomain.SentEmailSource) *RecordingEmailClientFactory {
	return &RecordingEmailClientFactory{inner: inner, recorder: recorder, source: source}
}

var _ messaging.EmailClientFactory = (*RecordingEmailClientFactory)(nil)

func (f *RecordingEmailClientFactory) NewClient(host, port, username, password, from string) messaging.Client {
	return NewRecordingEmailClient(f.inner.NewClient(host, port, username, password, from), f.recorder, f.source)
}
