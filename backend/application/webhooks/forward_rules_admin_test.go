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
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type fakeForwardRuleRepoAdmin struct {
	created []webhooks.ForwardRule
	rules   []webhooks.ForwardRule
	deleted []string
	err     error
}

func (f *fakeForwardRuleRepoAdmin) Create(ctx context.Context, r *webhooks.ForwardRule) error {
	if f.err != nil {
		return f.err
	}
	r.ID = "rule-1"
	f.created = append(f.created, *r)
	return nil
}
func (f *fakeForwardRuleRepoAdmin) List(ctx context.Context) ([]webhooks.ForwardRule, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rules, nil
}
func (f *fakeForwardRuleRepoAdmin) Delete(ctx context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeForwardRuleRepoAdmin) Match(ctx context.Context, provider, topic string) ([]webhooks.ForwardRule, error) {
	return nil, nil
}

func TestCreateForwardRuleUseCase_createsRuleUnderCallersTenant(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{}
	uc := appwebhooks.NewCreateForwardRuleUseCase(newMockGormDB(t, 1), repo)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	rule, err := uc.Execute(ctx, "woocommerce", "order.updated", "https://example.com/hook")

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "tenant-1", repo.created[0].TenantID)
	assert.Equal(t, "woocommerce", rule.Provider)
	assert.Equal(t, "order.updated", rule.Topic)
	assert.True(t, rule.Active)
}

func TestCreateForwardRuleUseCase_rejectsNonHTTPTargetURL(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{}
	uc := appwebhooks.NewCreateForwardRuleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), "woocommerce", "order.updated", "not-a-url")

	require.Error(t, err)
	assert.ErrorIs(t, err, webhooks.ErrInvalidTargetURL)
	assert.Empty(t, repo.created)
}

func TestCreateForwardRuleUseCase_rejectsFileSchemeTargetURL(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{}
	uc := appwebhooks.NewCreateForwardRuleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), "woocommerce", "order.updated", "file:///etc/passwd")

	require.Error(t, err)
	assert.ErrorIs(t, err, webhooks.ErrInvalidTargetURL)
}

func TestCreateForwardRuleUseCase_requiresProvider(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{}
	uc := appwebhooks.NewCreateForwardRuleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), "", "order.updated", "https://example.com/hook")

	require.Error(t, err)
	assert.Empty(t, repo.created)
}

func TestListForwardRulesUseCase_listsRules(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{rules: []webhooks.ForwardRule{{ID: "rule-1", Provider: "woocommerce"}}}
	uc := appwebhooks.NewListForwardRulesUseCase(newMockGormDB(t, 1), repo)

	rules, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "rule-1", rules[0].ID)
}

func TestDeleteForwardRuleUseCase_deletesRule(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{}
	uc := appwebhooks.NewDeleteForwardRuleUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "rule-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"rule-1"}, repo.deleted)
}

func TestDeleteForwardRuleUseCase_propagatesRepoError(t *testing.T) {
	repo := &fakeForwardRuleRepoAdmin{err: errors.New("db down")}
	uc := appwebhooks.NewDeleteForwardRuleUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "rule-1")

	require.Error(t, err)
}

type fakeForwardDeliveryRepoAdmin struct {
	recent []webhooks.ForwardDelivery
	err    error
}

func (f *fakeForwardDeliveryRepoAdmin) Create(ctx context.Context, d *webhooks.ForwardDelivery) error {
	return nil
}
func (f *fakeForwardDeliveryRepoAdmin) ListDue(ctx context.Context, now time.Time, limit int) ([]webhooks.ForwardDelivery, error) {
	return nil, nil
}
func (f *fakeForwardDeliveryRepoAdmin) MarkDelivered(ctx context.Context, id string) error {
	return nil
}
func (f *fakeForwardDeliveryRepoAdmin) MarkFailed(ctx context.Context, id string, errMsg string, nextAttemptAt time.Time, maxAttempts int) error {
	return nil
}
func (f *fakeForwardDeliveryRepoAdmin) ListRecent(ctx context.Context, limit int) ([]webhooks.ForwardDelivery, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.recent, nil
}

func TestListForwardDeliveriesUseCase_listsRecent(t *testing.T) {
	repo := &fakeForwardDeliveryRepoAdmin{recent: []webhooks.ForwardDelivery{{ID: "d-1"}}}
	uc := appwebhooks.NewListForwardDeliveriesUseCase(newMockGormDB(t, 1), repo)

	deliveries, err := uc.Execute(context.Background(), 50)

	require.NoError(t, err)
	require.Len(t, deliveries, 1)
}

func TestListForwardDeliveriesUseCase_defaultsOutOfRangeLimit(t *testing.T) {
	repo := &fakeForwardDeliveryRepoAdmin{}
	uc := appwebhooks.NewListForwardDeliveriesUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), 0)

	require.NoError(t, err)
}
