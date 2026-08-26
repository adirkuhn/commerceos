package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSalesOrderActivities_returnsRepoResult(t *testing.T) {
	activityRepo := &stubSalesOrderActivityRepo{listBySalesOrder: []omsDomain.SalesOrderActivity{{ID: "a-1", SalesOrderID: "so-1"}}}
	uc := appoms.NewListSalesOrderActivitiesUseCase(newMockGormDB(t), activityRepo)

	result, err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "a-1", result[0].ID)
}

func TestListUserActivities_returnsRepoResult(t *testing.T) {
	activityRepo := &stubSalesOrderActivityRepo{listByUser: []omsDomain.SalesOrderActivity{{ID: "a-1", UserID: "user-1"}}}
	uc := appoms.NewListUserActivitiesUseCase(newMockGormDB(t), activityRepo)

	result, err := uc.Execute(context.Background(), "user-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "user-1", result[0].UserID)
}
