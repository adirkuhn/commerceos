package assistant_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

var t0 = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func queued(t *testing.T) *assistant.Request {
	t.Helper()
	r, err := assistant.NewRequest("t-1", "u-1", "customer_admin", "status of order CO-12?", t0)
	require.NoError(t, err)
	return r
}

func TestNewRequest_isQueuedAndDueNow(t *testing.T) {
	r := queued(t)

	assert.Equal(t, assistant.StatusQueued, r.Status)
	assert.Equal(t, t0, r.NextAttemptAt)
	assert.Equal(t, "status of order CO-12?", r.Title)
	assert.Zero(t, r.Attempts)
}

func TestNewRequest_validatesText(t *testing.T) {
	_, err := assistant.NewRequest("t-1", "u-1", "customer_admin", "   ", t0)
	assert.ErrorIs(t, err, assistant.ErrTextRequired)

	_, err = assistant.NewRequest("t-1", "u-1", "customer_admin", string(make([]byte, assistant.MaxTextLength+1)), t0)
	assert.ErrorIs(t, err, assistant.ErrTextTooLong)
}

func TestNewRequest_titleIsFirstLineTruncated(t *testing.T) {
	long := "Hello team,\nplease quote 10 boxes"
	r, err := assistant.NewRequest("t-1", "u-1", "customer_admin", long, t0)
	require.NoError(t, err)
	assert.Equal(t, "Hello team,", r.Title)

	r, err = assistant.NewRequest("t-1", "u-1", "customer_admin", string(make([]rune, 200, 200)), t0)
	require.NoError(t, err)
	assert.LessOrEqual(t, len([]rune(r.Title)), assistant.MaxTitleLength)
}

func TestRequest_startCountsAttempt(t *testing.T) {
	r := queued(t)

	require.NoError(t, r.Start(t0))

	assert.Equal(t, assistant.StatusProcessing, r.Status)
	assert.Equal(t, 1, r.Attempts)
}

func TestRequest_startOnlyFromQueued(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Cancel(t0))

	assert.ErrorIs(t, r.Start(t0), assistant.ErrInvalidTransition)
}

func TestRequest_retryLaterBacksOffAndKeepsTheReason(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))

	require.NoError(t, r.RetryLater(assistant.WaitModelBusy, "429 rate limited", t0))

	assert.Equal(t, assistant.StatusQueued, r.Status)
	assert.Equal(t, assistant.WaitModelBusy, r.WaitReason)
	assert.Equal(t, "429 rate limited", r.LastError)
	assert.Equal(t, t0.Add(30*time.Second), r.NextAttemptAt)
}

func TestRetryDelay_doublesUpToTheCap(t *testing.T) {
	assert.Equal(t, 30*time.Second, assistant.RetryDelay(1))
	assert.Equal(t, 60*time.Second, assistant.RetryDelay(2))
	assert.Equal(t, 4*time.Minute, assistant.RetryDelay(4))
	assert.Equal(t, assistant.MaxRetryDelay, assistant.RetryDelay(6))
	assert.Equal(t, assistant.MaxRetryDelay, assistant.RetryDelay(500), "no overflow on a request waiting for days")
}

func TestRequest_answer(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))

	require.NoError(t, r.Answer(t0))

	assert.Equal(t, assistant.StatusAnswered, r.Status)
	assert.Empty(t, r.WaitReason)
	assert.Empty(t, r.LastError)
}

func TestRequest_askUserThenReplyRequeuesWithFreshBackoff(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))
	require.NoError(t, r.RetryLater(assistant.WaitModelBusy, "busy", t0))
	require.NoError(t, r.Start(t0))

	require.NoError(t, r.AskUser(t0))
	assert.Equal(t, assistant.StatusNeedsInput, r.Status)

	later := t0.Add(time.Hour)
	require.NoError(t, r.Reply(later))
	assert.Equal(t, assistant.StatusQueued, r.Status)
	assert.Equal(t, later, r.NextAttemptAt)
	assert.Zero(t, r.Attempts, "a new turn starts its own backoff")
}

func TestRequest_followUpAfterAnAnswer(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))
	require.NoError(t, r.Answer(t0))

	require.NoError(t, r.Reply(t0), "the user can ask a follow-up on an answered request")
	assert.Equal(t, assistant.StatusQueued, r.Status)
}

func TestRequest_replyNotWhileQueuedOrProcessing(t *testing.T) {
	r := queued(t)
	assert.ErrorIs(t, r.Reply(t0), assistant.ErrInvalidTransition)
	require.NoError(t, r.Start(t0))
	assert.ErrorIs(t, r.Reply(t0), assistant.ErrInvalidTransition)
}

func TestRequest_cancel(t *testing.T) {
	for _, setup := range []func(r *assistant.Request){
		func(r *assistant.Request) {},
		func(r *assistant.Request) { _ = r.Start(t0) },
		func(r *assistant.Request) { _ = r.Start(t0); _ = r.AskUser(t0) },
	} {
		r := queued(t)
		setup(r)
		require.NoError(t, r.Cancel(t0))
		assert.Equal(t, assistant.StatusCancelled, r.Status)
	}
}

func TestRequest_cancelNotOnceFinished(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))
	require.NoError(t, r.Answer(t0))
	assert.ErrorIs(t, r.Cancel(t0), assistant.ErrNotCancellable)

	r = queued(t)
	require.NoError(t, r.Cancel(t0))
	assert.ErrorIs(t, r.Cancel(t0), assistant.ErrNotCancellable)
}

func TestRequest_finishingAfterCancelIsRejected(t *testing.T) {
	// The worker may still be mid-call when the user cancels: whatever it
	// was about to record must not resurrect the request.
	r := queued(t)
	require.NoError(t, r.Start(t0))
	require.NoError(t, r.Cancel(t0))

	assert.ErrorIs(t, r.Answer(t0), assistant.ErrInvalidTransition)
	assert.ErrorIs(t, r.AskUser(t0), assistant.ErrInvalidTransition)
	assert.ErrorIs(t, r.RetryLater(assistant.WaitModelBusy, "", t0), assistant.ErrInvalidTransition)
}

func TestStatus_valid(t *testing.T) {
	assert.True(t, assistant.StatusNeedsInput.Valid())
	assert.False(t, assistant.Status("done").Valid())
}

func TestRequest_retryNowRunsAWaitingRequestNextWithFreshBackoff(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))
	require.NoError(t, r.RetryLater(assistant.WaitModelBusy, "429", t0))
	require.NoError(t, r.Start(t0.Add(time.Minute)))
	require.NoError(t, r.RetryLater(assistant.WaitModelBusy, "429", t0.Add(time.Minute)))
	later := t0.Add(2 * time.Minute)

	require.NoError(t, r.RetryNow(later))

	assert.Equal(t, assistant.StatusQueued, r.Status)
	assert.Equal(t, later, r.NextAttemptAt)
	assert.Zero(t, r.Attempts)
	assert.Equal(t, assistant.WaitModelBusy, r.WaitReason, "the last reason stays until the next attempt replaces it")
}

func TestRequest_retryNowOnlyWhileQueued(t *testing.T) {
	r := queued(t)
	require.NoError(t, r.Start(t0))
	assert.ErrorIs(t, r.RetryNow(t0), assistant.ErrInvalidTransition)
	require.NoError(t, r.Answer(t0))
	assert.ErrorIs(t, r.RetryNow(t0), assistant.ErrInvalidTransition)
}
