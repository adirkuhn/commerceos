package deadletters_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/deadletters"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// pagedStore serves each stream's dead letters `size` at a time, like the
// JetStream store's sequence cursor.
type pagedStore struct {
	byStream map[deadletter.Stream][]deadletter.DeadLetter
	size     int
	err      error
}

func (s *pagedStore) List(_ context.Context, f deadletter.Filter) (deadletter.Page, error) {
	if s.err != nil {
		return deadletter.Page{}, s.err
	}
	var rest []deadletter.DeadLetter
	for _, d := range s.byStream[f.Stream] {
		if d.Sequence > f.After {
			rest = append(rest, d)
		}
	}
	page := deadletter.Page{Items: rest}
	if len(rest) > s.size {
		page.Items = rest[:s.size]
		page.NextCursor = page.Items[s.size-1].Sequence
	}
	return page, nil
}

type sentNotification struct {
	tenantID   string
	superAdmin bool
	in         appnotifications.CreateNotificationInput
}

type stubNotifier struct {
	sent []sentNotification
	err  error
}

func (n *stubNotifier) Execute(ctx context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error) {
	n.sent = append(n.sent, sentNotification{tenantID: tenant.IDFromContext(ctx), superAdmin: tenant.IsSuperAdmin(ctx), in: in})
	return nil, n.err
}

var checkNow = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func waiting(stream deadletter.Stream, seq uint64, tenantID string, age time.Duration) deadletter.DeadLetter {
	at := checkNow.Add(-age)
	return deadletter.DeadLetter{Stream: stream, Sequence: seq, TenantID: tenantID, DeadLetteredAt: &at, StoredAt: at}
}

func TestCheckDeadLetters_alertsEachTenantAcrossBothStreamsAndPages(t *testing.T) {
	store := &pagedStore{size: 2, byStream: map[deadletter.Stream][]deadletter.DeadLetter{
		deadletter.StreamEvents: {
			waiting(deadletter.StreamEvents, 1, "t-1", time.Hour),
			waiting(deadletter.StreamEvents, 2, "t-2", 30*time.Hour),
			waiting(deadletter.StreamEvents, 3, "t-1", 2*time.Hour),
		},
		deadletter.StreamWebhooks: {waiting(deadletter.StreamWebhooks, 1, "t-1", 3*time.Hour)},
	}}
	notifier := &stubNotifier{}
	uc := deadletters.NewCheckDeadLettersUseCase(store, notifier, func() time.Time { return checkNow })

	require.NoError(t, uc.Run(tenant.WithSuperAdmin(context.Background())))

	require.Len(t, notifier.sent, 2)
	assert.Equal(t, sentNotification{tenantID: "t-1", in: appnotifications.CreateNotificationInput{
		Type: notificationsDomain.TypeDeadLettersWaiting, Params: map[string]any{"count": 3, "hours": 3},
	}}, notifier.sent[0], "only in its own tenant, never as super-admin")
	assert.Equal(t, sentNotification{tenantID: "t-2", in: appnotifications.CreateNotificationInput{
		Type: notificationsDomain.TypeDeadLettersStale, Params: map[string]any{"count": 1, "hours": 30},
	}}, notifier.sent[1], "older than deadletter.StaleAfter escalates")
}

func TestCheckDeadLetters_emptyQueueSendsNothing(t *testing.T) {
	notifier := &stubNotifier{}
	uc := deadletters.NewCheckDeadLettersUseCase(&pagedStore{size: 50}, notifier, func() time.Time { return checkNow })

	require.NoError(t, uc.Run(context.Background()))

	assert.Empty(t, notifier.sent)
}

// A dead letter without a tenant has no one to tell; it is only logged.
func TestCheckDeadLetters_withoutTenantSendsNothing(t *testing.T) {
	store := &pagedStore{size: 50, byStream: map[deadletter.Stream][]deadletter.DeadLetter{
		deadletter.StreamEvents: {waiting(deadletter.StreamEvents, 1, "", time.Hour)},
	}}
	notifier := &stubNotifier{}
	uc := deadletters.NewCheckDeadLettersUseCase(store, notifier, func() time.Time { return checkNow })

	require.NoError(t, uc.Run(context.Background()))

	assert.Empty(t, notifier.sent)
}

func TestCheckDeadLetters_storeErrorFailsTheRun(t *testing.T) {
	uc := deadletters.NewCheckDeadLettersUseCase(&pagedStore{err: errBoom}, &stubNotifier{}, func() time.Time { return checkNow })

	assert.ErrorIs(t, uc.Run(context.Background()), errBoom)
}

// One tenant's failed notification doesn't stop the others; the run still
// reports the failure.
func TestCheckDeadLetters_notifyFailureStillAlertsTheRest(t *testing.T) {
	store := &pagedStore{size: 50, byStream: map[deadletter.Stream][]deadletter.DeadLetter{
		deadletter.StreamEvents: {waiting(deadletter.StreamEvents, 1, "t-1", time.Hour), waiting(deadletter.StreamEvents, 2, "t-2", time.Hour)},
	}}
	notifier := &stubNotifier{err: errBoom}
	uc := deadletters.NewCheckDeadLettersUseCase(store, notifier, func() time.Time { return checkNow })

	err := uc.Run(context.Background())

	assert.ErrorIs(t, err, errBoom)
	assert.Len(t, notifier.sent, 2)
}
