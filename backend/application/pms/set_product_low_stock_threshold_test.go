package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubLowStockThresholdRepo struct {
	product *pmsDomain.Product
	setErr  error
	set     bool
}

func (s *stubLowStockThresholdRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Product, error) {
	if s.product == nil {
		return nil, pmsDomain.ErrProductNotFound
	}
	p := *s.product
	return &p, nil
}

func (s *stubLowStockThresholdRepo) SetLowStockThreshold(_ context.Context, _ string, threshold *int) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.set = true
	s.product.LowStockThreshold = threshold
	return nil
}


func TestSetProductLowStockThreshold_savesAndRecordsTheChange(t *testing.T) {
	repo := &stubLowStockThresholdRepo{product: &pmsDomain.Product{ID: "p1", LowStockThreshold: intPtr(10)}}
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewSetProductLowStockThresholdUseCase(newMockGormDB(t), repo, outbox)

	p, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "p1", intPtr(1000))

	require.NoError(t, err)
	require.Equal(t, 1000, *p.LowStockThreshold)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "pms.product.low_stock_threshold_changed", e.Subject)
	assert.Equal(t, "p1", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"low_stock_threshold": {float64(10), float64(1000)}}, e.Changes())
}

func TestSetProductLowStockThreshold_nilClearsTheOverride(t *testing.T) {
	repo := &stubLowStockThresholdRepo{product: &pmsDomain.Product{ID: "p1", LowStockThreshold: intPtr(5)}}
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewSetProductLowStockThresholdUseCase(newMockGormDB(t), repo, outbox)

	p, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "p1", nil)

	require.NoError(t, err)
	assert.Nil(t, p.LowStockThreshold)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, map[string][2]any{"low_stock_threshold": {float64(5), nil}}, e.Changes())
}

func TestSetProductLowStockThreshold_rejectsNegativeBeforeTouchingTheDatabase(t *testing.T) {
	repo := &stubLowStockThresholdRepo{product: &pmsDomain.Product{ID: "p1"}}
	uc := apppms.NewSetProductLowStockThresholdUseCase(nil, repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "p1", intPtr(-1))

	assert.ErrorIs(t, err, pmsDomain.ErrInvalidLowStockThreshold)
	assert.False(t, repo.set)
}

func TestSetProductLowStockThreshold_unknownProduct(t *testing.T) {
	uc := apppms.NewSetProductLowStockThresholdUseCase(newMockGormDB(t), &stubLowStockThresholdRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing", intPtr(3))

	assert.ErrorIs(t, err, pmsDomain.ErrProductNotFound)
}

func TestSetProductLowStockThreshold_failsWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	repo := &stubLowStockThresholdRepo{product: &pmsDomain.Product{ID: "p1"}}
	uc := apppms.NewSetProductLowStockThresholdUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{Err: boom})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "p1", intPtr(3))

	assert.ErrorIs(t, err, boom)
}
