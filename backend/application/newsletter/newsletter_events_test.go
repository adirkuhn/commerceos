package newsletter_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
)

func TestSubscribe_RecordsSubscribedWithoutTheEmail(t *testing.T) {
	w := newWorld(t)
	outbox := &eventrecordtest.Outbox{}
	w.deps.Outbox = outbox

	require.NoError(t, subscribe(w, "a@x.io"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "newsletter.subscriber.subscribed", e.Subject)
	assert.Equal(t, "s1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_channel_id": "chan-1", "resubscribed": false}, e.Data())
	raw, err := json.Marshal(e.Payload)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "a@x.io")
}

func TestUnsubscribe_RecordsUnsubscribed(t *testing.T) {
	w := newWorld(t)
	require.NoError(t, subscribe(w, "a@x.io"))
	s, _ := w.repo.FindByEmail(ctx, "a@x.io")
	outbox := &eventrecordtest.Outbox{}
	w.deps.Outbox = outbox

	require.NoError(t, appnewsletter.NewUnsubscribeUseCase(w.deps).Execute(ctx, s.UnsubscribeToken))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "newsletter.subscriber.unsubscribed", e.Subject)
	assert.Equal(t, s.ID, e.Payload["entity_id"])
}

func TestNewsletter_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	w := newWorld(t)
	w.deps.Outbox = &eventrecordtest.Outbox{Err: boom}

	assert.ErrorIs(t, subscribe(w, "a@x.io"), boom)
	assert.Empty(t, w.mail.sent, "no confirmation mail when the signup didn't commit")
}
