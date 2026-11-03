package dlq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/internal/idgen"
)

// maxScan bounds how many stored messages one List call reads while
// filtering; past it the page ends early with a cursor to continue from.
const maxScan = 2000

// streamNames maps each dead-letter stream to its JetStream stream.
var streamNames = map[deadletter.Stream]string{
	deadletter.StreamEvents:   publishers.DLQStreamName,
	deadletter.StreamWebhooks: publishers.WebhookDLQStreamName,
}

// Store reads, deletes and retries dead letters on JetStream. Reads use
// the stream message-get API, not a consumer, so reading never acks or
// removes anything.
type Store struct {
	js jetstream.JetStream
}

func NewStore(js jetstream.JetStream) *Store { return &Store{js: js} }

var (
	_ deadletter.Store          = (*Store)(nil)
	_ deadletter.RetryPublisher = (*Store)(nil)
)

func (s *Store) stream(ctx context.Context, which deadletter.Stream) (jetstream.Stream, error) {
	name, ok := streamNames[which]
	if !ok {
		return nil, deadletter.ErrInvalidStream
	}
	st, err := s.js.Stream(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("dlq: open stream %s: %w", name, err)
	}
	return st, nil
}

// List walks the stream from f.After by sequence, skipping deleted
// messages, until the page is full or maxScan messages were read.
func (s *Store) List(ctx context.Context, f deadletter.Filter) (deadletter.Page, error) {
	st, err := s.stream(ctx, f.Stream)
	if errors.Is(err, jetstream.ErrStreamNotFound) {
		return deadletter.Page{Items: []deadletter.DeadLetter{}}, nil
	}
	if err != nil {
		return deadletter.Page{}, err
	}
	state := st.CachedInfo().State
	page := deadletter.Page{Items: []deadletter.DeadLetter{}, Total: state.Msgs}
	limit := f.LimitOrDefault()
	seq := f.After + 1
	for scanned := 0; seq <= state.LastSeq; scanned++ {
		if scanned == maxScan {
			page.NextCursor = seq - 1
			return page, nil
		}
		raw, err := st.GetMsg(ctx, seq, jetstream.WithGetMsgSubject(">"))
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			break
		}
		if err != nil {
			return deadletter.Page{}, fmt.Errorf("dlq: read %s #%d: %w", f.Stream, seq, err)
		}
		seq = raw.Sequence + 1
		d := toDeadLetter(f.Stream, raw)
		if !f.Matches(d) {
			continue
		}
		page.Items = append(page.Items, d)
		if len(page.Items) == limit {
			if raw.Sequence < state.LastSeq {
				page.NextCursor = raw.Sequence
			}
			return page, nil
		}
	}
	return page, nil
}

func (s *Store) Get(ctx context.Context, which deadletter.Stream, seq uint64) (deadletter.DeadLetter, error) {
	st, err := s.stream(ctx, which)
	if err != nil {
		return deadletter.DeadLetter{}, notFoundIfMissing(err)
	}
	raw, err := st.GetMsg(ctx, seq)
	if err != nil {
		return deadletter.DeadLetter{}, notFoundIfMissing(fmt.Errorf("dlq: read %s #%d: %w", which, seq, err))
	}
	return toDeadLetter(which, raw), nil
}

func (s *Store) Delete(ctx context.Context, which deadletter.Stream, seq uint64) error {
	st, err := s.stream(ctx, which)
	if err != nil {
		return notFoundIfMissing(err)
	}
	err = st.DeleteMsg(ctx, seq)
	if errors.Is(err, jetstream.ErrMsgDeleteUnsuccessful) {
		// JetStream answers a delete of a message that is already gone with
		// this generic error, not "not found" -- check which it was.
		if _, getErr := st.GetMsg(ctx, seq); errors.Is(getErr, jetstream.ErrMsgNotFound) {
			err = getErr
		}
	}
	if err != nil {
		return notFoundIfMissing(fmt.Errorf("dlq: delete %s #%d: %w", which, seq, err))
	}
	return nil
}

// Republish publishes the dead letter's data on its original subject with
// a fresh Nats-Msg-Id (the original one may still be inside the stream's
// duplicate window); the event_id inside the data is unchanged, so each
// consumer's processed_events check stays authoritative.
func (s *Store) Republish(ctx context.Context, d deadletter.DeadLetter) error {
	if _, err := s.js.PublishMsg(ctx, retryMsg(d), jetstream.WithMsgID("retry-"+idgen.NewV4())); err != nil {
		return fmt.Errorf("dlq: republish %s: %w", d.Subject, err)
	}
	return nil
}

func notFoundIfMissing(err error) error {
	if errors.Is(err, jetstream.ErrMsgNotFound) || errors.Is(err, jetstream.ErrStreamNotFound) {
		return fmt.Errorf("%w: %w", deadletter.ErrNotFound, err)
	}
	return err
}

func retryMsg(d deadletter.DeadLetter) *nats.Msg {
	h := nats.Header{}
	if d.ConsumerKnown() {
		h.Set(HeaderRetryConsumer, d.Consumer)
	}
	return &nats.Msg{Subject: d.Subject, Data: d.Data, Header: h}
}

func toDeadLetter(which deadletter.Stream, raw *jetstream.RawStreamMsg) deadletter.DeadLetter {
	eventID, tenantID := deadletter.Identify(raw.Data)
	subject := raw.Header.Get(HeaderOriginalSubject)
	if subject == "" {
		subject = OriginalSubject(raw.Subject)
	}
	numDelivered, _ := strconv.Atoi(raw.Header.Get(HeaderNumDelivered)) // absent on old dead letters -> 0
	return deadletter.DeadLetter{
		Stream: which, Sequence: raw.Sequence, Subject: subject,
		Consumer: raw.Header.Get(HeaderConsumer), EventID: eventID, TenantID: tenantID,
		Error: raw.Header.Get(HeaderError), NumDelivered: numDelivered,
		DeadLetteredAt: parseTime(raw.Header.Get(HeaderDeadLetteredAt)),
		StoredAt:       raw.Time, Data: raw.Data,
	}
}

// parseTime is nil for a dead letter without (or with an unreadable)
// dead-lettered-at header.
func parseTime(v string) *time.Time {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}
	return &t
}
