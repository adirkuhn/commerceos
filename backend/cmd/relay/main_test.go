package main

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/outbox"
)

type stubRelayOutboxRepo struct {
	pending          []outbox.Message
	fetchErr         error
	published        []uint
	recordedFailures []uint
	recordFailureErr error
}

func (s *stubRelayOutboxRepo) FetchPending(_ context.Context, _ int) ([]outbox.Message, error) {
	return s.pending, s.fetchErr
}
func (s *stubRelayOutboxRepo) MarkPublished(_ context.Context, id uint) error {
	s.published = append(s.published, id)
	return nil
}
func (s *stubRelayOutboxRepo) RecordPublishFailure(_ context.Context, id uint, _ int) error {
	s.recordedFailures = append(s.recordedFailures, id)
	return s.recordFailureErr
}

type stubJSPublisher struct {
	calls []string
	// errBySubject, if set, fails a Publish for that subject with the given error.
	errBySubject map[string]error
}

func (s *stubJSPublisher) Publish(subj string, _ []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.calls = append(s.calls, subj)
	if err, ok := s.errBySubject[subj]; ok {
		return nil, err
	}
	return &nats.PubAck{}, nil
}

type stubConnChecker struct{ connected bool }

func (s stubConnChecker) IsConnected() bool { return s.connected }

func TestPublish_marksPublishedOnSuccess(t *testing.T) {
	repo := &stubRelayOutboxRepo{pending: []outbox.Message{{ID: 1, Subject: "pms.product.changed", Payload: []byte("{}")}}}
	js := &stubJSPublisher{}

	publish(context.Background(), repo, js, stubConnChecker{connected: true})

	assert.Equal(t, []uint{1}, repo.published)
	assert.Empty(t, repo.recordedFailures)
}

func TestPublish_skipsTheTickEntirelyWhenNotConnected(t *testing.T) {
	repo := &stubRelayOutboxRepo{pending: []outbox.Message{{ID: 1, Subject: "pms.product.changed"}}}
	js := &stubJSPublisher{}

	publish(context.Background(), repo, js, stubConnChecker{connected: false})

	assert.Empty(t, js.calls, "must not even attempt to publish while disconnected")
	assert.Empty(t, repo.published)
	assert.Empty(t, repo.recordedFailures, "a NATS outage must never burn a row's retry budget")
}

func TestPublish_connectivityErrorLeavesTheRowUntouched(t *testing.T) {
	repo := &stubRelayOutboxRepo{pending: []outbox.Message{{ID: 1, Subject: "pms.product.changed"}}}
	js := &stubJSPublisher{errBySubject: map[string]error{"pms.product.changed": nats.ErrNoResponders}}

	publish(context.Background(), repo, js, stubConnChecker{connected: true})

	assert.Empty(t, repo.published)
	assert.Empty(t, repo.recordedFailures, "a mid-batch connectivity drop must not count as a message failure")
}

func TestPublish_genuinePublishFailureRecordsIt(t *testing.T) {
	repo := &stubRelayOutboxRepo{pending: []outbox.Message{{ID: 1, Subject: "pms.product.changed"}}}
	js := &stubJSPublisher{errBySubject: map[string]error{"pms.product.changed": errors.New("max payload exceeded")}}

	publish(context.Background(), repo, js, stubConnChecker{connected: true})

	assert.Empty(t, repo.published)
	require.Equal(t, []uint{1}, repo.recordedFailures)
}

func TestPublish_continuesToLaterRowsAfterOneFails(t *testing.T) {
	repo := &stubRelayOutboxRepo{pending: []outbox.Message{
		{ID: 1, Subject: "bad.subject"},
		{ID: 2, Subject: "good.subject"},
	}}
	js := &stubJSPublisher{errBySubject: map[string]error{"bad.subject": errors.New("boom")}}

	publish(context.Background(), repo, js, stubConnChecker{connected: true})

	assert.Equal(t, []uint{1}, repo.recordedFailures)
	assert.Equal(t, []uint{2}, repo.published)
}

func TestPublish_fetchPendingErrorIsANoop(t *testing.T) {
	repo := &stubRelayOutboxRepo{fetchErr: errors.New("db down")}
	js := &stubJSPublisher{}

	publish(context.Background(), repo, js, stubConnChecker{connected: true})

	assert.Empty(t, js.calls)
}

func TestIsConnectivityErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"connection closed", nats.ErrConnectionClosed, true},
		{"timeout", nats.ErrTimeout, true},
		{"no responders", nats.ErrNoResponders, true},
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"wrapped timeout", errors.Join(errors.New("wrap"), nats.ErrTimeout), true},
		{"unrelated error", errors.New("max payload exceeded"), false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isConnectivityErr(tc.err))
		})
	}
}

// sanity check that the stub actually behaves like a *nats.Conn for our
// narrow connChecker interface, i.e. the interface is satisfied structurally.
var _ connChecker = stubConnChecker{}
