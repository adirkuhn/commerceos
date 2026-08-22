package oms_test

import (
	"context"
	"errors"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUpdateSalesOrder struct {
	// errByID, if set, fails Execute for that one order ID and succeeds for
	// every other -- proves a single bad order doesn't abort the batch.
	errByID  map[string]error
	gotCalls []struct {
		id string
		in appoms.UpdateSalesOrderInput
	}
}

func (s *stubUpdateSalesOrder) Execute(_ context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	s.gotCalls = append(s.gotCalls, struct {
		id string
		in appoms.UpdateSalesOrderInput
	}{id, in})
	if err, ok := s.errByID[id]; ok {
		return nil, err
	}
	return &omsDomain.SalesOrder{ID: id}, nil
}

func TestBulkUpdateSalesOrderStatus_appliesStatusToEveryOrderID(t *testing.T) {
	update := &stubUpdateSalesOrder{}
	uc := appoms.NewBulkUpdateSalesOrderStatusUseCase(update)
	status := omsDomain.SalesOrderStatusProcessing

	result, err := uc.Execute(context.Background(), appoms.BulkUpdateSalesOrderStatusInput{
		OrderIDs: []string{"so-1", "so-2", "so-3"}, Status: &status,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"so-1", "so-2", "so-3"}, result.Updated)
	assert.Empty(t, result.Failed)
	require.Len(t, update.gotCalls, 3)
	for _, call := range update.gotCalls {
		require.NotNil(t, call.in.Status)
		assert.Equal(t, status, *call.in.Status)
	}
}

func TestBulkUpdateSalesOrderStatus_appliesFulfilledToggle(t *testing.T) {
	update := &stubUpdateSalesOrder{}
	uc := appoms.NewBulkUpdateSalesOrderStatusUseCase(update)
	fulfilled := true

	_, err := uc.Execute(context.Background(), appoms.BulkUpdateSalesOrderStatusInput{
		OrderIDs: []string{"so-1"}, Fulfilled: &fulfilled,
	})

	require.NoError(t, err)
	require.Len(t, update.gotCalls, 1)
	require.NotNil(t, update.gotCalls[0].in.Fulfilled)
	assert.True(t, *update.gotCalls[0].in.Fulfilled)
}

func TestBulkUpdateSalesOrderStatus_oneFailureDoesNotAbortTheRest(t *testing.T) {
	update := &stubUpdateSalesOrder{errByID: map[string]error{"so-2": omsDomain.ErrSalesOrderNotFound}}
	uc := appoms.NewBulkUpdateSalesOrderStatusUseCase(update)
	status := omsDomain.SalesOrderStatusCancelled

	result, err := uc.Execute(context.Background(), appoms.BulkUpdateSalesOrderStatusInput{
		OrderIDs: []string{"so-1", "so-2", "so-3"}, Status: &status,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"so-1", "so-3"}, result.Updated)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "so-2", result.Failed[0].OrderID)
	assert.Contains(t, result.Failed[0].Error, "not found")
	// every order still gets attempted, not just up to the first failure
	require.Len(t, update.gotCalls, 3)
}

func TestBulkUpdateSalesOrderStatus_emptyOrderIDsIsRejected(t *testing.T) {
	uc := appoms.NewBulkUpdateSalesOrderStatusUseCase(&stubUpdateSalesOrder{})
	status := omsDomain.SalesOrderStatusCancelled

	_, err := uc.Execute(context.Background(), appoms.BulkUpdateSalesOrderStatusInput{Status: &status})

	require.Error(t, err)
	assert.True(t, errors.Is(err, omsDomain.ErrBulkUpdateOrderIDsRequired))
}

func TestBulkUpdateSalesOrderStatus_noFieldsGivenIsRejected(t *testing.T) {
	uc := appoms.NewBulkUpdateSalesOrderStatusUseCase(&stubUpdateSalesOrder{})

	_, err := uc.Execute(context.Background(), appoms.BulkUpdateSalesOrderStatusInput{OrderIDs: []string{"so-1"}})

	require.Error(t, err)
	assert.True(t, errors.Is(err, omsDomain.ErrBulkUpdateNoFieldsGiven))
}
