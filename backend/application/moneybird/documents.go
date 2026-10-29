package moneybird

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// documentRecorder stores where a send ended -- synced, or failed with the
// reason -- and raises the matching event.
type documentRecorder struct {
	db     *gorm.DB
	docs   mb.DocumentRepository
	outbox eventrecord.Enqueuer
}

func (r documentRecorder) find(ctx context.Context, sourceType mb.SourceType, sourceID string) (*mb.Document, error) {
	var doc *mb.Document
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		var err error
		doc, err = r.docs.Find(ctx, sourceType, sourceID)
		return err
	})
	return doc, err
}

// synced records the document as booked.
func (r documentRecorder) synced(ctx context.Context, doc mb.Document) (*mb.Document, error) {
	doc.Status, doc.Error, doc.SyncedAt = mb.DocumentSynced, "", time.Now()
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		if err := r.docs.Save(ctx, &doc); err != nil {
			return err
		}
		return eventrecord.Record(ctx, r.outbox, events.MoneybirdDocumentSynced, doc.SourceID, eventrecord.Data{
			"source_type": string(doc.SourceType), "moneybird_id": doc.MoneybirdID, "reference": doc.Reference,
			"payment_registered": doc.PaymentRegistered,
		})
	})
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// failed records why the send failed and returns cause. MoneybirdID stays
// when the document was already created, so a retry resumes it.
func (r documentRecorder) failed(ctx context.Context, doc mb.Document, cause error) error {
	doc.Status, doc.Error, doc.SyncedAt = mb.DocumentFailed, cause.Error(), time.Now()
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		if err := r.docs.Save(ctx, &doc); err != nil {
			return err
		}
		return eventrecord.Record(ctx, r.outbox, events.MoneybirdDocumentSyncFailed, doc.SourceID, eventrecord.Data{
			"source_type": string(doc.SourceType), "moneybird_id": doc.MoneybirdID, "reference": doc.Reference,
			"error": doc.Error,
		})
	})
	if err != nil {
		return errors.Join(cause, fmt.Errorf("recording the failure: %w", err))
	}
	return cause
}

// documentLines maps booked lines onto Moneybird tax rates and the
// default ledger account.
func documentLines(s mb.Settings, booked []mb.BookedLine) ([]mb.DocumentLine, error) {
	out := make([]mb.DocumentLine, len(booked))
	for i, b := range booked {
		taxRateID, err := s.TaxRateID(b.RateBasisPoints)
		if err != nil {
			return nil, fmt.Errorf("%w (line %q)", err, b.Description)
		}
		out[i] = mb.DocumentLine{
			Description: b.Description, Amount: b.Amount, PriceMinorUnits: b.PriceMinorUnits,
			TaxRateID: taxRateID, LedgerAccountID: s.DefaultLedgerAccountID,
		}
	}
	return out, nil
}
