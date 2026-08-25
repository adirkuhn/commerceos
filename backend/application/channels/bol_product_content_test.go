package channels_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/outbox"
)

// fakeOutboxRepo records every enqueued outbox row -- used here to prove
// UpsertBolProductContentUseCase raises the same pms.product.changed event
// a normal product edit does (mandatory outbox pattern, see CLAUDE.md),
// atomically with the content write.
type fakeOutboxRepo struct {
	enqueued []fakeOutboxRow
	err      error
}

type fakeOutboxRow struct {
	Subject    string
	EventID    string
	Payload    []byte
	OccurredAt time.Time
}

func (f *fakeOutboxRepo) Enqueue(_ context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.enqueued = append(f.enqueued, fakeOutboxRow{Subject: subject, EventID: eventID, Payload: payload, OccurredAt: occurredAt})
	return nil
}
func (f *fakeOutboxRepo) FetchPending(context.Context, int) ([]outbox.Message, error) {
	return nil, nil
}
func (f *fakeOutboxRepo) MarkPublished(context.Context, uint) error { return nil }
func (f *fakeOutboxRepo) MarkFailed(context.Context, uint) error    { return nil }

func TestGetBolProductContent_returnsNilWhenNoneExists(t *testing.T) {
	repo := &fakeBolContentRepo{}
	uc := appchannels.NewGetBolProductContentUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestUpsertBolProductContent_savesFields(t *testing.T) {
	repo := &fakeBolContentRepo{}
	outboxRepo := &fakeOutboxRepo{}
	uc := appchannels.NewUpsertBolProductContentUseCase(newMockGormDB(t, 1), repo, outboxRepo)
	brand := "Hipoglós"
	widthMM := int32(80)

	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductContentInput{
		ProductID: "prod-1", ConnectionID: "conn-1", Brand: &brand, PackageWidthMM: &widthMM,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Brand)
	assert.Equal(t, "Hipoglós", *result.Brand)
	require.Len(t, repo.upserted, 1)
	require.NotNil(t, repo.upserted[0].PackageWidthMM)
	assert.Equal(t, int32(80), *repo.upserted[0].PackageWidthMM)
	assert.Equal(t, "prod-1", repo.upserted[0].ProductID)
}

// TestUpsertBolProductContent_raisesProductChangedEvent is the test the
// user asked for: saving product content under the Bol tab must raise a
// real pms.product.changed domain event (via the outbox), the same
// mechanism a normal product edit uses to update every channel the
// product is linked to -- not just the frontend's own direct
// pushProductToBol call, which only covers Bol and gives no durability/
// retry if it fails mid-flight.
func TestUpsertBolProductContent_raisesProductChangedEvent(t *testing.T) {
	repo := &fakeBolContentRepo{}
	outboxRepo := &fakeOutboxRepo{}
	uc := appchannels.NewUpsertBolProductContentUseCase(newMockGormDB(t, 1), repo, outboxRepo)

	_, err := uc.Execute(context.Background(), appchannels.UpsertBolProductContentInput{
		ProductID: "prod-1", ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, outboxRepo.enqueued, 1)
	row := outboxRepo.enqueued[0]
	assert.Equal(t, "pms.product.changed", row.Subject)
	assert.NotEmpty(t, row.EventID)

	var event events.ProductChanged
	require.NoError(t, json.Unmarshal(row.Payload, &event))
	assert.Equal(t, "prod-1", event.ProductID)
	assert.Equal(t, row.EventID, event.EventID)
}

func TestUpsertBolProductContent_propagatesOutboxEnqueueFailure(t *testing.T) {
	repo := &fakeBolContentRepo{}
	outboxRepo := &fakeOutboxRepo{err: assert.AnError}
	uc := appchannels.NewUpsertBolProductContentUseCase(newMockGormDB(t, 1), repo, outboxRepo)

	_, err := uc.Execute(context.Background(), appchannels.UpsertBolProductContentInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.Error(t, err, "the content write must not silently succeed while the event fails to enqueue -- both happen in the same transaction")
}
