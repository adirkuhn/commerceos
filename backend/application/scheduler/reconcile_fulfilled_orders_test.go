package scheduler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubReconcileSalesOrderRepo struct {
	listResult  []omsDomain.SalesOrder
	listErr     error
	byID        map[string]omsDomain.SalesOrder
	findErrByID map[string]error
	listFilter  omsDomain.ListSalesOrdersFilter
}

func (s *stubReconcileSalesOrderRepo) Create(context.Context, *omsDomain.SalesOrder) error {
	return nil
}
func (s *stubReconcileSalesOrderRepo) FindByID(_ context.Context, id string) (*omsDomain.SalesOrder, error) {
	if err, ok := s.findErrByID[id]; ok {
		return nil, err
	}
	o, ok := s.byID[id]
	if !ok {
		return nil, omsDomain.ErrSalesOrderNotFound
	}
	return &o, nil
}
func (s *stubReconcileSalesOrderRepo) FindByExternalID(context.Context, omsDomain.SalesOrderSource, string) (*omsDomain.SalesOrder, error) {
	return nil, nil
}
func (s *stubReconcileSalesOrderRepo) Search(_ context.Context, _ string, _ int) ([]omsDomain.SalesOrder, error) {
	return nil, nil
}
func (s *stubReconcileSalesOrderRepo) List(_ context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	s.listFilter = filter
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listResult, int64(len(s.listResult)), nil
}
func (s *stubReconcileSalesOrderRepo) ListByConnection(context.Context, string) ([]omsDomain.SalesOrder, error) {
	return nil, nil
}
func (s *stubReconcileSalesOrderRepo) Update(context.Context, string, map[string]any) error {
	return nil
}
func (s *stubReconcileSalesOrderRepo) TrySetStockDecremented(context.Context, string, bool, bool) (bool, error) {
	return false, nil
}
func (s *stubReconcileSalesOrderRepo) Delete(context.Context, string) error { return nil }

type stubReconcileUpdater struct {
	calls []struct {
		id     string
		status omsDomain.SalesOrderStatus
	}
	err error
}

func (s *stubReconcileUpdater) Execute(_ context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	if s.err != nil {
		return nil, s.err
	}
	var status omsDomain.SalesOrderStatus
	if in.Status != nil {
		status = *in.Status
	}
	s.calls = append(s.calls, struct {
		id     string
		status omsDomain.SalesOrderStatus
	}{id: id, status: status})
	return &omsDomain.SalesOrder{ID: id}, nil
}

func TestReconcileFulfilledOrders_movesStuckOrdersToShipped(t *testing.T) {
	orderA := omsDomain.SalesOrder{ID: "order-a", TenantID: "tenant-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true}
	orderB := omsDomain.SalesOrder{ID: "order-b", TenantID: "tenant-2", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusPending, Fulfilled: true}
	repo := &stubReconcileSalesOrderRepo{
		listResult: []omsDomain.SalesOrder{orderA, orderB},
		byID:       map[string]omsDomain.SalesOrder{"order-a": orderA, "order-b": orderB},
	}
	updater := &stubReconcileUpdater{}
	uc := appscheduler.NewReconcileFulfilledOrdersUseCase(newMockGormDBN(t, 3), repo, updater)

	err := uc.Run(context.Background())

	require.NoError(t, err)
	require.Len(t, updater.calls, 2)
	assert.Equal(t, "order-a", updater.calls[0].id)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
	assert.Equal(t, "order-b", updater.calls[1].id)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[1].status, "internal status is uniformly Shipped -- Woo's lack of a native shipped status is handled at the channel-push boundary, not here")

	fulfilled := true
	assert.Equal(t, &fulfilled, repo.listFilter.Fulfilled)
	assert.ElementsMatch(t, []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing}, repo.listFilter.Status)
}

func TestReconcileFulfilledOrders_oneOrderFailing_doesNotBlockTheRest(t *testing.T) {
	orderA := omsDomain.SalesOrder{ID: "order-a", TenantID: "tenant-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true}
	orderB := omsDomain.SalesOrder{ID: "order-b", TenantID: "tenant-2", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true}
	repo := &stubReconcileSalesOrderRepo{
		listResult:  []omsDomain.SalesOrder{orderA, orderB},
		byID:        map[string]omsDomain.SalesOrder{"order-b": orderB},
		findErrByID: map[string]error{"order-a": errors.New("db exploded")},
	}
	updater := &stubReconcileUpdater{}
	uc := appscheduler.NewReconcileFulfilledOrdersUseCase(newMockGormDBN(t, 3), repo, updater)

	err := uc.Run(context.Background())

	require.NoError(t, err, "one order's failure is logged, not fatal to the run")
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-b", updater.calls[0].id)
}

func TestReconcileFulfilledOrders_listError_failsTheRun(t *testing.T) {
	repo := &stubReconcileSalesOrderRepo{listErr: errors.New("db down")}
	uc := appscheduler.NewReconcileFulfilledOrdersUseCase(newMockGormDBN(t, 1), repo, &stubReconcileUpdater{})

	err := uc.Run(context.Background())

	require.Error(t, err)
}
