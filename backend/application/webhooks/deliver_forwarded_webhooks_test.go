package webhooks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

type fakeForwardDeliveryQueueRepo struct {
	due          []webhooks.ForwardDelivery
	delivered    []string
	failed       []string
	failedErrMsg map[string]string
	listErr      error
}

func (f *fakeForwardDeliveryQueueRepo) Create(ctx context.Context, d *webhooks.ForwardDelivery) error {
	return nil
}
func (f *fakeForwardDeliveryQueueRepo) ListDue(ctx context.Context, now time.Time, limit int) ([]webhooks.ForwardDelivery, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.due, nil
}
func (f *fakeForwardDeliveryQueueRepo) MarkDelivered(ctx context.Context, id string) error {
	f.delivered = append(f.delivered, id)
	return nil
}
func (f *fakeForwardDeliveryQueueRepo) MarkFailed(ctx context.Context, id string, errMsg string, nextAttemptAt time.Time, maxAttempts int) error {
	f.failed = append(f.failed, id)
	if f.failedErrMsg == nil {
		f.failedErrMsg = map[string]string{}
	}
	f.failedErrMsg[id] = errMsg
	return nil
}
func (f *fakeForwardDeliveryQueueRepo) ListRecent(ctx context.Context, limit int) ([]webhooks.ForwardDelivery, error) {
	return nil, nil
}

type fakePoster struct {
	statusByURL map[string]int
	errByURL    map[string]error
	calls       []struct {
		url  string
		body []byte
	}
}

func (f *fakePoster) Post(ctx context.Context, targetURL string, contentType string, body []byte) (int, error) {
	f.calls = append(f.calls, struct {
		url  string
		body []byte
	}{targetURL, body})
	if f.errByURL != nil {
		if err, ok := f.errByURL[targetURL]; ok {
			return 0, err
		}
	}
	if f.statusByURL != nil {
		if status, ok := f.statusByURL[targetURL]; ok {
			return status, nil
		}
	}
	return 200, nil
}

func TestDeliverForwardedWebhooksUseCase_deliversDueRows(t *testing.T) {
	repo := &fakeForwardDeliveryQueueRepo{due: []webhooks.ForwardDelivery{
		{ID: "d-1", TenantID: "tenant-1", TargetURL: "https://example.com/hook", Payload: []byte(`{"id":1}`)},
	}}
	poster := &fakePoster{}
	uc := appwebhooks.NewDeliverForwardedWebhooksUseCase(newMockGormDB(t, 2), repo, poster)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Checked)
	assert.Equal(t, 1, result.Delivered)
	assert.Empty(t, result.Errors)
	assert.Equal(t, []string{"d-1"}, repo.delivered)
	require.Len(t, poster.calls, 1)
	assert.Equal(t, "https://example.com/hook", poster.calls[0].url)
	assert.Equal(t, []byte(`{"id":1}`), poster.calls[0].body)
}

func TestDeliverForwardedWebhooksUseCase_nonSuccessStatus_marksFailed(t *testing.T) {
	repo := &fakeForwardDeliveryQueueRepo{due: []webhooks.ForwardDelivery{
		{ID: "d-1", TenantID: "tenant-1", TargetURL: "https://example.com/hook", Attempts: 0},
	}}
	poster := &fakePoster{statusByURL: map[string]int{"https://example.com/hook": 500}}
	uc := appwebhooks.NewDeliverForwardedWebhooksUseCase(newMockGormDB(t, 2), repo, poster)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, result.Delivered)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "d-1")
	assert.Equal(t, []string{"d-1"}, repo.failed)
	assert.Contains(t, repo.failedErrMsg["d-1"], "500")
}

func TestDeliverForwardedWebhooksUseCase_transportError_marksFailed(t *testing.T) {
	repo := &fakeForwardDeliveryQueueRepo{due: []webhooks.ForwardDelivery{
		{ID: "d-1", TenantID: "tenant-1", TargetURL: "https://example.com/hook"},
	}}
	poster := &fakePoster{errByURL: map[string]error{"https://example.com/hook": errors.New("connection refused")}}
	uc := appwebhooks.NewDeliverForwardedWebhooksUseCase(newMockGormDB(t, 2), repo, poster)

	_, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"d-1"}, repo.failed)
	assert.Contains(t, repo.failedErrMsg["d-1"], "connection refused")
}

func TestDeliverForwardedWebhooksUseCase_noDueRows_noOp(t *testing.T) {
	repo := &fakeForwardDeliveryQueueRepo{}
	poster := &fakePoster{}
	uc := appwebhooks.NewDeliverForwardedWebhooksUseCase(newMockGormDB(t, 1), repo, poster)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.Checked)
	assert.Empty(t, poster.calls)
}

func TestDeliverForwardedWebhooksUseCase_listDueError_propagates(t *testing.T) {
	repo := &fakeForwardDeliveryQueueRepo{listErr: errors.New("db down")}
	poster := &fakePoster{}
	uc := appwebhooks.NewDeliverForwardedWebhooksUseCase(newMockGormDB(t, 1), repo, poster)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}

func TestDeliverForwardedWebhooksUseCase_partialFailure_continuesRestOfBatch(t *testing.T) {
	repo := &fakeForwardDeliveryQueueRepo{due: []webhooks.ForwardDelivery{
		{ID: "d-1", TenantID: "tenant-1", TargetURL: "https://bad.example.com/hook"},
		{ID: "d-2", TenantID: "tenant-1", TargetURL: "https://good.example.com/hook"},
	}}
	poster := &fakePoster{errByURL: map[string]error{"https://bad.example.com/hook": errors.New("timeout")}}
	uc := appwebhooks.NewDeliverForwardedWebhooksUseCase(newMockGormDB(t, 3), repo, poster)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 1, result.Delivered)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, []string{"d-1"}, repo.failed)
	assert.Equal(t, []string{"d-2"}, repo.delivered)
}
