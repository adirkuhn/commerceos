package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestListSalesOrderOrigins_returnsThemBySalesOrder(t *testing.T) {
	repo := &fakeOriginRepo{rows: []integrationsDomain.SalesOrderOrigin{
		{SalesOrderID: "so-1", Origin: integrationsDomain.OriginWooCommerce, ExternalOrderID: "1001"},
		{SalesOrderID: "so-2", Origin: integrationsDomain.OriginPOS, ExternalOrderID: "pos-1"},
		{SalesOrderID: "so-3", Origin: integrationsDomain.OriginManual},
	}}

	got, err := appintegrations.NewListSalesOrderOriginsUseCase(newMockGormDB(t), repo).Execute(context.Background(), []string{"so-1", "so-2"})

	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "1001", got["so-1"].ExternalOrderID)
	assert.Equal(t, integrationsDomain.OriginPOS, got["so-2"].Origin)
}

func TestListSalesOrderOrigins_noOrdersNeedsNoQuery(t *testing.T) {
	got, err := appintegrations.NewListSalesOrderOriginsUseCase(nil, &fakeOriginRepo{}).Execute(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestListSalesOrderOrigins_aReadFailureIsReturned(t *testing.T) {
	boom := errors.New("db down")
	_, err := appintegrations.NewListSalesOrderOriginsUseCase(newMockGormDB(t), &fakeOriginRepo{findErr: boom}).Execute(context.Background(), []string{"so-1"})

	assert.ErrorIs(t, err, boom)
}
